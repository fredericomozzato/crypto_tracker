require 'rails_helper'

RSpec.describe AssetsHelper, type: :helper do
  describe '.asset_value' do
    it 'returns formatted value in USD' do
      coin   = create :coin, rate: 909.9
      amount = 2.5

      expect(asset_value(coin, amount)).to eq '$2,274.75'
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
