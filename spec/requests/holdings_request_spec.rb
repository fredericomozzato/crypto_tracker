require 'rails_helper'

RSpec.describe '/holdings', type: :request do
  describe 'POST /portfolio/holdings' do
    context 'authenticated' do
      it 'creates a holding associated with the portfolio' do
        coin = create :coin, name: 'Coin', ticker: 'COI', rate: 9.99
        user = create :user
        portfolio = create :portfolio, account: user.account
        params = { holding: { coin_id: coin.id, portfolio_id: portfolio.id } }

        login_as user, scope: :user
        post(portfolio_holdings_path(portfolio), params:)

        expect(response).to redirect_to portfolio_path(portfolio)
        expect(flash[:notice]).to eq 'COI added to portfolio'
        expect(Holding.count).to eq 1
        expect(portfolio.holdings.count).to eq 1
        expect(portfolio.holdings.last.coin).to eq coin
        expect(portfolio.holdings.last.amount).to eq 0.0
      end

      it 'creates a holding with a specified initial amount' do
        coin = create :coin, name: 'Coin', rate: 9.99
        user = create :user
        portfolio = create :portfolio, account: user.account
        params = { holding: { coin_id: coin.id, amount: 10, portfolio_id: portfolio.id } }

        login_as user, scope: :user
        post(portfolio_holdings_path(portfolio), params:)

        expect(portfolio.holdings.last.amount).to eq 10.0
      end

      it 'can\'t create a holding if portfolio already has one with the same coin' do
        coin = create :coin, name: 'Coin'
        user = create :user
        portfolio = create :portfolio, account: user.account
        holding = create :holding, coin:, portfolio:, amount: 9.99
        params = { holding: { coin_id: coin.id, amount: 1.11, portfolio_id: portfolio.id } }

        login_as user, scope: :user
        post(portfolio_holdings_path(portfolio), params:)

        expect(flash[:alert]).to eq 'Can\'t add coin to portfolio'
        expect(portfolio.holdings.count).to eq 1
        expect(portfolio.holdings.last).to eq holding
      end
    end

    context 'unauthenticated' do
      it 'redirects to login page and doesn\'t create holding' do
        coin = create :coin
        portfolio = create :portfolio
        params = { holding: { coin_id: coin.id, portfolio_id: portfolio.id } }

        post(portfolio_holdings_path(portfolio), params:)

        expect(response).to redirect_to new_user_session_path
        expect(portfolio.holdings.count).to eq 0
      end
    end
  end

  describe 'PATCH /holding/:id' do
    context 'authenticated' do
      it 'Deposit valid amount with success' do
        coin = create :coin, ticker: 'COI'
        portfolio = create :portfolio
        holding = create :holding, portfolio:, coin:, amount: 10.0
        deposit_amount = 5.0
        params = { holding: { id: holding.id,
                              operation: 'deposit',
                              amount: deposit_amount } }

        login_as portfolio.owner, scope: :user
        patch(holding_path(holding), params:)

        expect(response).to redirect_to portfolio_path(portfolio)
        expect(flash[:notice]).to eq 'Deposited 5.0 COI to portfolio'
        expect(holding.reload.amount).to eq 15.0
      end

      it 'Can\'t deposit negative amount' do
        portfolio = create :portfolio
        holding = create :holding, portfolio:, amount: 5.0
        deposit_amount = -1.0
        params = { holding: { id: holding.id,
                              operation: 'deposit',
                              amount: deposit_amount } }

        login_as portfolio.owner, scope: :user
        patch(holding_path(holding), params:)

        expect(holding.amount).to eq 5.0
      end

      it 'Withdraws valid amount with success' do
        coin = create :coin, ticker: 'COI'
        portfolio = create :portfolio
        holding = create :holding, portfolio:, coin:, amount: 10.0
        withdraw_amount = 5.5
        params = { holding: { id: holding.id,
                              operation: 'withdraw',
                              amount: withdraw_amount } }

        login_as portfolio.owner, scope: :user
        patch(holding_path(holding), params:)

        expect(response).to redirect_to portfolio_path(portfolio)
        expect(flash[:notice]).to eq 'Withdrew 5.5 COI from portfolio'
        expect(holding.reload.amount).to eq 4.5
      end

      it 'Can\'t withdraw negative amount' do
        portfolio = create :portfolio
        holding = create :holding, portfolio:, amount: 10.0
        withdraw_amount = -5.5
        params = { holding: { id: holding.id,
                              operation: 'withdraw',
                              amount: withdraw_amount } }

        login_as portfolio.owner, scope: :user
        patch(holding_path(holding), params:)

        expect(holding.reload.amount).to eq 10
      end

      it 'Can\'t withdraw more than the holding\'s amount' do
        portfolio = create :portfolio
        holding = create :holding, portfolio:, amount: 10.0
        withdraw_amount = 10.1
        params = { holding: { id: holding.id,
                              operation: 'withdraw',
                              amount: withdraw_amount } }

        login_as portfolio.owner, scope: :user
        patch(holding_path(holding), params:)

        expect(holding.reload.amount).to eq 10
      end

      it 'Updates holding\'s amount with success' do
        coin = create :coin, ticker: 'COI'
        portfolio = create :portfolio
        holding = create :holding, portfolio:, coin:, amount: 10.0
        new_amount = 5.5
        params = { holding: { id: holding.id,
                              operation: 'update',
                              amount: new_amount } }

        login_as portfolio.owner, scope: :user
        patch(holding_path(holding), params:)

        expect(response).to redirect_to portfolio_path(portfolio)
        expect(flash[:notice]).to eq 'Updated COI to 5.5'
        expect(holding.reload.amount).to eq 5.5
      end

      it 'Update holding amount to 0' do
        portfolio = create :portfolio
        holding = create :holding, portfolio:, amount: 10.0
        new_amount = 0.0
        params = { holding: { id: holding.id,
                              operation: 'update',
                              amount: new_amount } }

        login_as portfolio.owner, scope: :user
        patch(holding_path(holding), params:)

        expect(holding.reload.amount).to eq 0.0
      end

      it 'Can\'t update holding to negative amount' do
        portfolio = create :portfolio
        holding = create :holding, portfolio:, amount: 5.5
        new_amount = -0.1
        params = { holding: { id: holding.id,
                              operation: 'update',
                              amount: new_amount } }

        login_as portfolio.owner, scope: :user
        patch(holding_path(holding), params:)

        expect(holding.reload.amount).to eq 5.5
      end
    end

    context 'unauthenticated' do
      it 'Redirects to login page and doesn\'t modify holding' do
        portfolio = create :portfolio
        holding = create :holding, portfolio:, amount: 5.0
        deposit_amount = -1.0
        params = { holding: { id: holding.id,
                              operation: 'deposit',
                              amount: deposit_amount } }

        patch(holding_path(holding), params:)

        expect(response).to redirect_to new_user_session_path
        expect(holding.amount).to eq 5.0
      end
    end
  end

  describe 'DELETE /holding/:id' do
    context 'authenticated' do
      it 'Deletes holding from portfolio' do
        coin_a = create :coin, ticker: 'CNA'
        coin_b = create :coin, ticker: 'CNB'
        portfolio = create :portfolio
        holding_a = create :holding, portfolio:, coin: coin_a
        holding_b = create :holding, portfolio:, coin: coin_b

        login_as portfolio.owner, scope: :user
        delete(holding_path(holding_a))

        expect(portfolio.holdings).not_to include holding_a
        expect(portfolio.holdings).to include holding_b
        expect(response).to redirect_to portfolio_path(portfolio)
        expect(flash[:notice]).to eq 'Removed CNA from portfolio'
      end
    end
  end
end
