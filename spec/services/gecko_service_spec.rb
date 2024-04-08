require 'rails_helper'

RSpec.describe GeckoService do
  describe '.top_markets' do
    it 'Calls CoinGecko API\'s markets endpoint' do
      connection_spy = spy Faraday::Connection
      stub_const 'Faraday::Connection', connection_spy

      GeckoService.top_markets

      expect(connection_spy).to have_received(:get).with(
        '/api/v3/coins/markets', {
          vs_currency: 'usd',
          order: 'market_cap_desc',
          per_page: 100,
          price_change_percentage: '24h',
          precision: 8
        }
      )
    end
  end

  describe '.prices' do
    it 'Calls Coin Gecko API\'s price endpoint with api_id for databse coins' do
      create :coin, api_id: 'coin_a'
      create :coin, api_id: 'coin_b'
      create :coin, api_id: 'coin_c'

      connection_spy = spy Faraday::Connection
      stub_const 'Faraday::Connection', connection_spy

      GeckoService.prices

      expect(connection_spy).to have_received(:get).with(
        '/api/v3/simple/price',
        ids: 'coin_a,coin_b,coin_c',
        vs_currencies: 'usd',
        include_24hr_change: true,
        precision: 8
      )
    end
  end
end
