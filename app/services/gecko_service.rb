class GeckoService
  def self.top_markets
    conn = Faraday.new(url: 'https://api.coingecko.com',
                       headers: { 'Content-Type': 'application/json' })

    res = conn.get('/api/v3/coins/markets', { vs_currency: 'usd',
                                              order: 'market_cap_desc',
                                              per_page: 100,
                                              price_change_percentage: '24h',
                                              precision: 8 })

    res.body if res.status == 200
  end

  def self.prices; end
end
