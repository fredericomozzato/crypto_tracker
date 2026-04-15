package ui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredericomozzato/crypto_tracker/internal/api"
	"github.com/fredericomozzato/crypto_tracker/internal/format"
	"github.com/fredericomozzato/crypto_tracker/internal/store"
)

// MarketsModel manages the Markets tab: coin list, auto-refresh, cursor, status bar.
type MarketsModel struct {
	width            int
	height           int
	ctx              context.Context
	store            store.Store
	client           api.CoinGeckoClient
	coins            []store.Coin
	lastErr          string
	refreshing       bool
	lastRefreshed    time.Time
	cursor           int
	offset           int
	rateLimitedUntil time.Time // time at which rate-limit cooldown expires
	refreshAttempts  int       // consecutive rate-limit errors (for backoff)
}

// coinsLoadedMsg is sent when coins are successfully loaded from the API.
type coinsLoadedMsg struct {
	coins []store.Coin
}

// errMsg is sent when an error occurs during data fetching.
type errMsg struct {
	err error
}

// pricesUpdatedMsg is sent when prices are successfully refreshed.
type pricesUpdatedMsg struct {
	coins []store.Coin
}

// tickMsg fires every 5 seconds from cmdTick.
type tickMsg time.Time

const (
	coinFetchLimit  = 100
	refreshInterval = 60 * time.Second
	staleThreshold  = 5 * time.Minute
)

// NewMarketsModel creates a new MarketsModel with the given dependencies.
func NewMarketsModel(ctx context.Context, s store.Store, c api.CoinGeckoClient) MarketsModel {
	return MarketsModel{
		ctx:    ctx,
		store:  s,
		client: c,
	}
}

// Init returns the batched load + tick commands.
func (m MarketsModel) Init() tea.Cmd {
	return tea.Batch(m.cmdLoad(), cmdTick())
}

// cmdTick returns a command that fires a tickMsg after 5 seconds.
func cmdTick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// cmdLoad returns a command that loads coins from the database or fetches from API.
func (m MarketsModel) cmdLoad() tea.Cmd {
	return func() tea.Msg {
		existing, err := m.store.GetAllCoins(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading coins: %w", err)}
		}
		if len(existing) >= coinFetchLimit {
			return coinsLoadedMsg{coins: existing}
		}

		fetched, err := m.client.FetchMarkets(m.ctx, coinFetchLimit)
		if err != nil {
			return errMsg{err: err}
		}
		for _, c := range fetched {
			if err := m.store.UpsertCoin(m.ctx, c); err != nil {
				return errMsg{err: fmt.Errorf("upserting coin %s: %w", c.ApiID, err)}
			}
		}
		stored, err := m.store.GetAllCoins(m.ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("loading coins after seed: %w", err)}
		}
		return coinsLoadedMsg{coins: stored}
	}
}

// update handles all markets messages. Returns typed MarketsModel, not tea.Model.
// Does NOT handle 'q' or Ctrl+C — those belong to AppModel.
func (m MarketsModel) update(msg tea.Msg) (MarketsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				switch r {
				case 'r':
					if !m.refreshing && time.Now().After(m.rateLimitedUntil) && len(m.coins) > 0 {
						m.refreshing = true
						return m, m.cmdRefresh()
					}
				case 'j':
					m.moveCursor(+1)
				case 'k':
					m.moveCursor(-1)
				case 'g':
					m.cursor = 0
					m.adjustViewport()
				case 'G':
					if len(m.coins) > 0 {
						m.cursor = len(m.coins) - 1
						m.adjustViewport()
					}
				}
			}
		case tea.KeyDown:
			m.moveCursor(+1)
		case tea.KeyUp:
			m.moveCursor(-1)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		cmds := []tea.Cmd{cmdTick()}
		now := time.Now()
		canRefresh := !m.refreshing && len(m.coins) > 0 && now.Sub(m.lastRefreshed) >= refreshInterval && now.After(m.rateLimitedUntil)
		if canRefresh {
			m.refreshing = true
			cmds = append(cmds, m.cmdRefresh())
		}
		return m, tea.Batch(cmds...)
	case coinsLoadedMsg:
		m.coins = msg.coins
		m.lastErr = ""
		m.lastRefreshed = time.Now()
		if m.cursor >= len(m.coins) && len(m.coins) > 0 {
			m.cursor = len(m.coins) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	case pricesUpdatedMsg:
		m.coins = msg.coins
		m.refreshing = false
		m.lastErr = ""
		m.lastRefreshed = time.Now()
		m.refreshAttempts = 0            // reset backoff on success
		m.rateLimitedUntil = time.Time{} // clear rate-limit state
	case errMsg:
		m.refreshing = false
		if api.IsRateLimitError(msg.err) {
			backoff := api.RetryAfterFromError(msg.err, api.DefaultRetryAfter)
			multiplier := 1 << min(m.refreshAttempts, 3) // 1, 2, 4, 8 (capped)
			cooldown := backoff * time.Duration(multiplier)
			if cooldown > 5*time.Minute {
				cooldown = 5 * time.Minute
			}
			m.rateLimitedUntil = time.Now().Add(cooldown)
			m.refreshAttempts++
			m.lastErr = "" // status bar shows rate-limit state, not raw error
		} else {
			m.lastErr = msg.err.Error()
		}
	}

	return m, nil
}

// cmdRefresh returns a command that refreshes prices for all loaded coins.
func (m MarketsModel) cmdRefresh() tea.Cmd {
	return func() tea.Msg {
		apiIDs := make([]string, len(m.coins))
		for i, c := range m.coins {
			apiIDs[i] = c.ApiID
		}

		prices, err := m.client.FetchPrices(m.ctx, apiIDs)
		if err != nil {
			return errMsg{err: err}
		}

		if err := m.store.UpdatePrices(m.ctx, prices); err != nil {
			return errMsg{err: err}
		}

		updatedCoins, err := m.store.GetAllCoins(m.ctx)
		if err != nil {
			return errMsg{err: err}
		}

		return pricesUpdatedMsg{coins: updatedCoins}
	}
}

// InputActive always returns false — Markets has no text inputs.
func (m MarketsModel) InputActive() bool {
	return false
}

// View renders the coin table + status bar. Assumes height set via WindowSizeMsg.
func (m MarketsModel) View() string {
	h := m.tableHeight()
	end := m.offset + h
	if end > len(m.coins) {
		end = len(m.coins)
	}

	if len(m.coins) == 0 {
		return "loading...\n" + m.renderStatusBar()
	}

	wRank := 4
	wName := 22
	wTicker := 8
	wPrice := 14
	wChange := 9

	highlight := lipgloss.NewStyle().Reverse(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	header := fmt.Sprintf(
		"%*s  %-*s  %-*s  %*s  %*s",
		wRank, "#",
		wName, "Name",
		wTicker, "Ticker",
		wPrice, "Price (USD)",
		wChange, "24h",
	)

	var lines []string
	lines = append(lines, header)

	for i := m.offset; i < end; i++ {
		c := m.coins[i]
		price := format.FmtPrice(c.Rate)
		change := format.FmtChange(c.PriceChange)

		if c.PriceChange >= 0 {
			change = green.Render(change)
		} else {
			change = red.Render(change)
		}

		line := fmt.Sprintf(
			"%*d  %-*s  %-*s  %*s  %*s",
			wRank, c.MarketRank,
			wName, truncate(c.Name, wName-2),
			wTicker, c.Ticker,
			wPrice, price,
			wChange, change,
		)

		if i == m.cursor {
			line = highlight.Render(line)
		}

		lines = append(lines, line)
	}

	for len(lines)-1 < h {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n") + "\n" + m.renderStatusBar()
}

// statusRight returns the right-hand portion of the status bar.
func (m MarketsModel) statusRight() string {
	now := time.Now()
	if m.refreshing {
		return "Refreshing"
	}
	if now.Before(m.rateLimitedUntil) {
		secs := int(time.Until(m.rateLimitedUntil).Seconds())
		return fmt.Sprintf("Rate limited — retry in %ds", secs)
	}
	if m.lastErr != "" {
		return "error: " + m.lastErr
	}
	if m.lastRefreshed.IsZero() {
		return "loading..."
	}
	if now.Sub(m.lastRefreshed) > staleThreshold {
		return "Stale"
	}
	return "Synced"
}

// renderStatusBar returns a two-sided status bar with hints on the left and
// sync status on the right.
func (m MarketsModel) renderStatusBar() string {
	leftContent := "j/k navigate • g/G top/bottom • r refresh • q quit"
	rightContent := m.statusRight()

	grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))

	rateLimitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00")) // dark orange

	var rightStyled string
	switch {
	case strings.HasPrefix(rightContent, "Rate limited"):
		rightStyled = rateLimitStyle.Render(rightContent)
	case rightContent == "Synced":
		rightStyled = greenStyle.Render(rightContent)
	case rightContent == "Stale":
		rightStyled = yellowStyle.Render(rightContent)
	case strings.HasPrefix(rightContent, "error:"):
		rightStyled = errStyle.Render(rightContent)
	default:
		rightStyled = grayStyle.Render(rightContent)
	}

	leftStyled := grayStyle.Render(leftContent)
	padding := m.width - lipgloss.Width(leftContent) - lipgloss.Width(rightContent)
	if padding < 1 {
		padding = 1
	}
	return leftStyled + strings.Repeat(" ", padding) + rightStyled
}

// moveCursor moves the cursor by delta and adjusts the viewport.
func (m *MarketsModel) moveCursor(delta int) {
	if len(m.coins) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.coins) {
		m.cursor = len(m.coins) - 1
	}
	m.adjustViewport()
}

// adjustViewport updates m.offset so the cursor row stays visible.
func (m *MarketsModel) adjustViewport() {
	h := m.tableHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	maxOff := len(m.coins) - h
	if maxOff < 0 {
		maxOff = 0
	}
	if m.offset > maxOff {
		m.offset = maxOff
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// tableHeight returns the number of rows available for coin data.
// Reserves 1 row for column headers and 1 row for the status bar.
func (m MarketsModel) tableHeight() int {
	h := m.height - 2
	if h < 1 {
		return 1
	}
	return h
}

// truncate returns s truncated to maxLen characters with an ellipsis.
func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	runes := []rune(s)
	return string(runes[:maxLen-1]) + "…"
}
