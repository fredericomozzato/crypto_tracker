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

    it 'Save coins not yet present in the database and doesn\'t touch the others' do
      btc = create :coin, name: 'Bitcoin', api_id: 'bitcoin', ticker: 'BTC',
                          icon: 'https://assets.coingecko.com/coins/images/1/large/bitcoin.png',
                          rate: 70_042.55900708
      eth = create :coin, name: 'Ethereum', api_id: 'ethereum', ticker: 'ETH',
                          icon: 'https://assets.coingecko.com/coins/images/279/large/ethereum.png',
                          rate: 3_571.72987487
      fake_res = File.read(File.join(__dir__, '..', 'support', 'json', 'coin_markets.json'))

      allow(GeckoService).to receive(:top_markets).and_return fake_res
      CoinService.import_coins

      expect(Coin.count).to eq 5
      expect(btc.reload.changed?).to be false
      expect(eth.reload.changed?).to be false
    end
  end

  describe '.refresh_rates' do
    it 'Updates the rate of every Coin in the database' do
      btc  = create :coin, api_id: 'bitcoin',     rate: 0.0
      eth  = create :coin, api_id: 'ethereum',    rate: 0.0
      usdt = create :coin, api_id: 'tether',      rate: 0.0
      bnb  = create :coin, api_id: 'binancecoin', rate: 0.0
      sol  = create :coin, api_id: 'solana',      rate: 0.0
      prices_json = File.read(File.join(__dir__, '..', 'support', 'json', 'coin_prices.json'))

      allow(GeckoService).to receive(:prices).and_return prices_json
      CoinService.refresh_rates

      expect(btc.reload.rate).to  eq 68_650.94737118
      expect(eth.reload.rate).to  eq 3_510.27391192
      expect(usdt.reload.rate).to eq 0.99531325
      expect(bnb.reload.rate).to  eq 565.41173079
      expect(sol.reload.rate).to  eq 181.80816183
    end
  end
end
