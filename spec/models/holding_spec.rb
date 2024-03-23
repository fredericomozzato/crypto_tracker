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
end
