require 'rails_helper'

RSpec.describe Coin, type: :model do
  describe '#valid?' do
    it 'false without name' do
      coin = build :coin, name: ''

      expect(coin).not_to be_valid
      expect(coin.errors).to include :name
      expect(coin.errors.full_messages).to include 'Name can\'t be blank'
    end

    it 'false with non-unique name' do
      coin = create :coin, name: 'Coin A'
      invalid_coin = build :coin, name: coin.name

      expect(invalid_coin).not_to be_valid
      expect(invalid_coin.errors).to include :name
      expect(invalid_coin.errors.full_messages).to include 'Name has already been taken'
    end

    it 'false without API ID' do
      coin = build :coin, api_id: ''

      expect(coin).not_to be_valid
      expect(coin.errors).to include :api_id
      expect(coin.errors.full_messages).to include 'API ID can\'t be blank'
    end

    it 'false with non-unique API ID' do
      coin = create :coin, api_id: 'coin_a'
      invalid_coin = build :coin, api_id: coin.api_id

      expect(invalid_coin).not_to be_valid
      expect(invalid_coin.errors).to include :api_id
      expect(invalid_coin.errors.full_messages).to include 'API ID has already been taken'
    end

    it 'false without ticker' do
      coin = build :coin, ticker: ''

      expect(coin).not_to be_valid
      expect(coin.errors).to include :ticker
      expect(coin.errors.full_messages).to include 'Ticker can\'t be blank'
    end

    it 'false with non-unique ticker' do
      coin = create :coin, ticker: 'CNA'
      invalid_coin = build :coin, ticker: coin.ticker

      expect(invalid_coin).not_to be_valid
      expect(invalid_coin.errors).to include :ticker
      expect(invalid_coin.errors.full_messages).to include 'Ticker has already been taken'
    end

    it 'false without icon' do
      coin = build :coin, icon: ''

      expect(coin).not_to be_valid
      expect(coin.errors).to include :icon
      expect(coin.errors.full_messages).to include 'Icon can\'t be blank'
    end

    it 'false with non-unique icon' do
      coin = create :coin, icon: 'coin_a.jpg'
      invalid_coin = build :coin, icon: coin.icon

      expect(invalid_coin).not_to be_valid
      expect(invalid_coin.errors).to include :icon
      expect(invalid_coin.errors.full_messages).to include 'Icon has already been taken'
    end

    it 'false with negative rate' do
      coin = build :coin, rate: -1

      expect(coin).not_to be_valid
      expect(coin.errors).to include :rate
      expect(coin.errors.full_messages).to include 'Rate must be greater than or equal to 0'
    end
  end
end
