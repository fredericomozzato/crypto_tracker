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
end
