require 'rails_helper'

RSpec.describe '/portfolios', type: :request do
  describe 'POST /portfolios' do
    context 'authenticated' do
      it 'User creates a new portfolio with success' do
        user = create :user
        params = { portfolio: { name: 'Test Portfolio' } }

        login_as user, scope: :user
        post(portfolios_path, params:)

        expect(response).to redirect_to portfolios_path
        expect(flash[:notice]).to eq 'Portfolio successfully created'
        expect(Portfolio.count).to eq 1
        expect(Portfolio.last.name).to eq 'Test Portfolio'
        expect(Portfolio.last.account).to eq user.account
      end

      it 'User can\'t create a portfolio without name' do
        user = create :user
        params = { portfolio: { name: '' } }

        login_as user, scope: :user
        post(portfolios_path, params:)

        expect(flash[:alert]).to eq 'Unable to create Portfolio'
        expect(Portfolio.count).to eq 0
      end
    end

    context 'not authenticated' do
      it 'doesn\'t create Portfolio and is redirected to the login page' do
        params = { portfolio: { name: 'Test Portfolio' } }

        post(portfolios_path, params:)

        expect(response).to redirect_to new_user_session_path
        expect(Portfolio.count).to eq 0
      end
    end
  end

  describe 'GET /portfolios' do
    context 'authenticated' do
      it 'returns 200 OK' do
        user = create :user

        login_as user, scope: :user
        get portfolios_path

        expect(response).to have_http_status :ok
      end
    end

    context 'not authenticated' do
      it 'redirects to login page' do
        get portfolios_path

        expect(response).to redirect_to new_user_session_path
      end
    end
  end

  describe 'DELETE /portfolios/:id' do
    context 'authenticated' do
      it 'deletes a specific portfolio' do
        user = create :user
        portfolio1 = create :portfolio, name: 'P1', account: user.account
        portfolio2 = create :portfolio, name: 'P2', account: user.account

        login_as user, scope: :user
        delete portfolio_path(portfolio1)

        expect(response).to redirect_to portfolios_path
        expect(flash[:notice]).to eq 'Portfolio deleted successfully'
        expect(user.portfolios.count).to eq 1
        expect(user.portfolios).to include portfolio2
        expect(user.portfolios).not_to include portfolio1
      end
    end

    context 'not authenticated' do
      it 'redirects to login page and doesn\'t delete portfolio' do
        portfolio = create :portfolio

        delete portfolio_path(portfolio)

        expect(response).to redirect_to new_user_session_path
        expect(Portfolio.last).to eq portfolio
      end
    end
  end
end
