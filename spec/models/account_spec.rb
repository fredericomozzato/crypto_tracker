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

  describe '#net_worth' do
    it 'returns the total USD value of the account' do
      coin_a = create :coin, rate: 6.66
      coin_b = create :coin, rate: 5.55
      coin_c = create :coin, rate: 4.44
      account = create(:user).account
      portfolio1 = create(:portfolio, account:)
      portfolio1.holdings.create([{ coin: coin_a, amount: 2 },
                                  { coin: coin_b, amount: 3 },
                                  { coin: coin_c, amount: 4 }])
      portfolio2 = create(:portfolio, account:)
      portfolio2.holdings.create([{ coin: coin_a, amount: 5 },
                                  { coin: coin_b, amount: 6 },
                                  { coin: coin_c, amount: 7 }])
      portfolio3 = create(:portfolio, account:)
      portfolio3.holdings.create([{ coin: coin_a, amount: 1 },
                                  { coin: coin_b, amount: 2 },
                                  { coin: coin_c, amount: 3 }])

      expect(account.net_worth).to eq 176.49
    end

    it 'returns 0 if the account has no portoflios' do
      account = create(:user).account

      expect(account.net_worth).to eq 0
    end
  end

  describe '#assets' do
    it 'Returns aggregated list of all the account\'s assets and their amounts' do
      coin_a = create :coin, ticker: 'CNA', rate: 1.11
      coin_b = create :coin, ticker: 'CNB', rate: 2.22
      coin_c = create :coin, ticker: 'CNC', rate: 3.33
      coin_d = create :coin, ticker: 'CND', rate: 4.44
      user = create :user
      portfolio1 = create :portfolio, account: user.account
      portfolio2 = create :portfolio, account: user.account
      portfolio1.holdings.create([{ coin: coin_a, amount: 5.5 },
                                  { coin: coin_b, amount: 6.6 },
                                  { coin: coin_c, amount: 7.7 }])
      portfolio2.holdings.create([{ coin: coin_b, amount: 2.2 },
                                  { coin: coin_c, amount: 3.3 },
                                  { coin: coin_d, amount: 4.4 }])
      other_user = create :user
      other_portfolio = create :portfolio, account: other_user.account
      other_portfolio.holdings.create([{ coin: coin_a, amount: 900 },
                                       { coin: coin_b, amount: 500 },
                                       { coin: coin_c, amount: 600 }])

      assets = user.account.assets

      expect(assets.count).to   eq 4
      expect(assets[coin_a]).to eq 5.5
      expect(assets[coin_b]).to eq 8.8
      expect(assets[coin_c]).to eq 11
      expect(assets[coin_d]).to eq 4.4
    end

    it 'return an empty hash if there are no assets in account' do
      coin_a = create :coin, ticker: 'CNA', rate: 1.11
      coin_b = create :coin, ticker: 'CNB', rate: 2.22
      coin_c = create :coin, ticker: 'CNC', rate: 3.33
      user = create :user
      other_user = create :user
      other_portfolio = create :portfolio, account: other_user.account
      other_portfolio.holdings.create([{ coin: coin_a, amount: 900 },
                                       { coin: coin_b, amount: 500 },
                                       { coin: coin_c, amount: 600 }])

      assets = user.account.assets

      expect(assets).to eq({})
    end
  end
end
