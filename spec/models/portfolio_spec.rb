require 'rails_helper'

RSpec.describe Portfolio, type: :model do
  describe '#valid?' do
    it 'false without name' do
      portfolio = build :portfolio, name: ''

      expect(portfolio).not_to be_valid
      expect(portfolio.errors).to include :name
      expect(portfolio.errors.full_messages).to include 'Name can\'t be blank'
    end
  end

  describe '#total_balance' do
    it 'returns the total value in USD for the portfolio' do
      coin_a = create :coin, rate: 9.99
      coin_b = create :coin, rate: 8.88
      coin_c = create :coin, rate: 7.77
      portfolio = create :portfolio
      portfolio.holdings.create([{ coin: coin_a, amount: 1 },
                                 { coin: coin_b, amount: 2 },
                                 { coin: coin_c, amount: 3 }])

      expect(portfolio.total_balance).to eq 51.06
    end

    it 'returns 0 if there are no holdings in the portfolio' do
      portfolio = create :portfolio

      expect(portfolio.total_balance).to eq 0
    end
  end
end
