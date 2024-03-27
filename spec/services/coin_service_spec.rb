require 'rails_helper'

RSpec.describe CoinService do
  describe '.import_coins' do
    it 'Save coins to the database' do
      fake_res = File.read(File.join(__dir__, '..', 'support', 'json', 'coin_markets.json'))

      allow(GeckoService).to receive(:top_markets).and_return fake_res
      CoinService.import_coins

      expect(Coin.count).to eq 5
      expect(Coin.all[0].name).to eq 'Bitcoin'
      expect(Coin.all[1].name).to eq 'Ethereum'
      expect(Coin.all[2].name).to eq 'Tether'
      expect(Coin.all[3].name).to eq 'BNB'
      expect(Coin.all[4].name).to eq 'Solana'
    end
  end
end
