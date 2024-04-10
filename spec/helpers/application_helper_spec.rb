require 'rails_helper'

RSpec.describe ApplicationHelper, type: :helper do
  describe '.balance_format' do
    it 'Returns original balance in the hundreds' do
      balance = 123.45

      expect(balance_format(balance)).to eq '$123.45'
    end

    it 'Returns abbreviated balance in the thousands with K' do
      balance = 1_234.56

      expect(balance_format(balance)).to eq '$1.23K'
    end

    it 'Returns abbreviated balance in the millions with M' do
      balance = 12_345_678.99

      expect(balance_format(balance)).to eq '$12.35M'
    end

    it 'Returns abbreviated balance in the billions with B' do
      balance = 1_234_567_899.99

      expect(balance_format(balance)).to eq '$1.23B'
    end
  end

  describe '.asset_percentage' do
    it 'returns the proprtion of an asset in an account' do
      coin_a = create :coin, rate: 5.0
      coin_b = create :coin, rate: 10.0
      coin_c = create :coin, rate: 15.0
      account = create(:user).account
      portfolio = create(:portfolio, account:)
      holding_a = create :holding, portfolio:, coin: coin_a, amount: 1.0
      holding_b = create :holding, portfolio:, coin: coin_b, amount: 2.0
      holding_c = create :holding, portfolio:, coin: coin_c, amount: 3.0

      percentage_a = asset_percentage(coin_a, holding_a.amount, account.net_worth)
      percentage_b = asset_percentage(coin_b, holding_b.amount, account.net_worth)
      percentage_c = asset_percentage(coin_c, holding_c.amount, account.net_worth)

      expect(percentage_a).to eq '7.14%'
      expect(percentage_b).to eq '28.57%'
      expect(percentage_c).to eq '64.29%'
    end
  end
end
