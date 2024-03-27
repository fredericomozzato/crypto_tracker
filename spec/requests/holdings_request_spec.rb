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

    context 'not authenticated' do
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
end
