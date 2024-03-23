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
end
