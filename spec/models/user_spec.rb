require 'rails_helper'

RSpec.describe User, type: :model do
  describe '#valid' do
    it 'false without email' do
      user = build :user, email: ''

      expect(user).not_to be_valid
      expect(user.errors).to include :email
      expect(user.errors.full_messages).to include 'Email can\'t be blank'
    end

    it 'false with non-unique email' do
      user = create :user, email: 'user@email.com'
      invalid_user = build :user, email: user.email

      expect(invalid_user).not_to be_valid
      expect(invalid_user.errors).to include :email
      expect(invalid_user.errors.full_messages).to include 'Email has already been taken'
    end

    it 'false without password' do
      user = build :user, password: ''

      expect(user).not_to be_valid
      expect(user.errors).to include :password
      expect(user.errors.full_messages).to include 'Password can\'t be blank'
    end
  end

  describe '#create' do
    it 'creates an account associated with the user' do
      user = create :user

      expect(user.account).not_to be_nil
      expect(Account.last.owner).to eq user
    end
  end
end
