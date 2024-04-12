require 'rails_helper'

RSpec.describe Holding, type: :model do
  describe '#valid?' do
    it 'false with negative amount' do
      holding = build :holding, amount: -1

      expect(holding).not_to be_valid
      expect(holding.errors).to include :amount
      expect(holding.errors.full_messages).to include 'Amount must be greater than or equal to 0'
    end

    it 'false if coin already in portfolio' do
      coin = create :coin
      portfolio = create :portfolio
      portfolio.holdings.create({ coin: })
      invalid_holding = build(:holding, portfolio:, coin:)

      expect(invalid_holding).not_to be_valid
      expect(invalid_holding.errors).to include :portfolio
      expect(invalid_holding.errors.full_messages).to include 'Portfolio already has a holding with this coin'
    end
  end

  describe '#value' do
    it 'returns the value in USD' do
      coin = create :coin, rate: 9.99
      holding = build :holding, coin:, amount: 5.55

      expect(holding.value).to eq 55.4445
    end
  end

  describe '#deposit' do
    it 'Adds amount to the holding' do
      holding = build :holding, amount: 10.0

      holding.deposit 5.55

      expect(holding.amount).to eq 15.55
    end

    it 'Doesn\'t accept negative amount' do
      holding = create :holding, amount: 5.0

      holding.deposit(-1)

      expect(holding.amount).to eq 5.0
    end
  end

  describe '#withdraw' do
    it 'Withdraws amount from holding' do
      holding = build :holding, amount: 10.0

      holding.withdraw 9.99

      expect(holding.amount).to eq 0.01
    end

    it 'Doesn\'t accept negative values' do
      holding = build :holding, amount: 5.0

      holding.withdraw(-2.5)

      expect(holding.amount).to eq 5.0
    end

    it 'Doesn\'t accept amount greater than the holding\'s amount' do
      holding = build :holding, amount: 10.0

      holding.withdraw 10.1

      expect(holding.amount).to eq 10.0
    end

    it 'Can withdraw the exact amount' do
      holding = build :holding, amount: 12.3456789

      holding.withdraw 12.3456789

      expect(holding.amount).to eq 0.0
    end
  end

  describe '#proportion' do
    it 'Returns the holding\'s USD proportion in relation to portfolio total value' do
      coin_a    = create :coin, rate: 1.11
      coin_b    = create :coin, rate: 2.22
      coin_c    = create :coin, rate: 3.33
      portfolio = create :portfolio
      holding_a = create :holding, portfolio:, coin: coin_a, amount: 5
      holding_b = create :holding, portfolio:, coin: coin_b, amount: 4
      holding_c = create :holding, portfolio:, coin: coin_c, amount: 3

      expect(holding_a.proportion).to eq 22.73
      expect(holding_b.proportion).to eq 36.36
      expect(holding_c.proportion).to eq 40.91
    end

    it 'Returns 0.0 if portfolio has no balance' do
      coin_a    = create :coin, rate: 1.11
      portfolio = create :portfolio
      holding_a = create :holding, portfolio:, coin: coin_a, amount: 0

      expect(holding_a.proportion).to eq 0.0
    end
  end
end
