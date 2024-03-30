require 'rails_helper'

RSpec.describe '/account', type: :request do
  describe 'GET /account' do
    context 'authenticated and authorized' do
      it 'returns 200 OK' do
        user = create :user

        login_as user, scope: :user
        get account_path(user.account)

        expect(response).to have_http_status :ok
      end
    end

    context 'unauthenticated' do
      it 'redirects to login page' do
        user = create :user

        get account_path(user.account)

        expect(response).to redirect_to new_user_session_path
      end
    end
  end
end
