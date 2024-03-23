require 'rails_helper'

RSpec.describe Account, type: :model do
  describe '#valid?' do
    it 'false without UUID' do
      user = create :user
      acc = Account.new owner: user
      allow(SecureRandom).to receive(:uuid).and_return ''

      expect(acc).not_to be_valid
      expect(acc.errors).to include :uuid
      expect(acc.errors.full_messages).to include 'UUID can\'t be blank'
    end

    it 'false with non-unique UUID' do
      uuid = 'ebd71e8d-5d16-4ec7-a7c1-3d451b87521d'
      allow(SecureRandom).to receive(:uuid).and_return uuid
      create :user
      user = build :user
      allow(SecureRandom).to receive(:uuid).and_return uuid
      acc = Account.new owner: user

      expect(acc).not_to be_valid
      expect(acc.errors).to include :uuid
      expect(acc.errors.full_messages).to include 'UUID has already been taken'
    end

    it 'false if owner already has another account' do
      user = create :user
      acc = Account.new owner: user

      expect(acc).not_to be_valid
      expect(acc.errors).to include :owner
      expect(acc.errors.full_messages).to include 'Owner already has an account'
    end
  end
end
