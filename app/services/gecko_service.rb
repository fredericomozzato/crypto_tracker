class GeckoService
  GECKO_CONSTANTS = Rails.configuration.coingecko_api

  def self.conn
    Faraday.new(url: GECKO_CONSTANTS[:base_url],
                headers: { 'Content-Type': 'application/json' })
  end
  private_class_method :conn

  def self.top_markets
    res = conn.get(GECKO_CONSTANTS[:markets_url], {
                     vs_currency: 'usd',
                     order: 'market_cap_desc',
                     per_page: 100,
                     price_change_percentage: '24h',
                     precision: 8
                   })

    res.body if res.status == 200
  end

  def self.prices
    res = conn.get(GECKO_CONSTANTS[:prices_url],
                   { ids: Coin.ids_as_string,
                     vs_currencies: GECKO_CONSTANTS[:supported_currencies],
                     precision: 8 })

    res.body if res.status == 200
  end
end
