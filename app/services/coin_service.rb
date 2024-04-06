class CoinService
  def self.import_coins
    res = JSON.parse GeckoService.top_markets, symbolize_names: true
    res.each { |row| Coin.create coin_params row }
  end

  def self.refresh_rates
    res = JSON.parse GeckoService.prices, symbolize_names: true

    res.each_pair do |api_id, current_price|
      coin = Coin.find_by(api_id:)
      coin&.update rate: current_price[:usd]
    end
  end

  def self.coin_params(row)
    {
      name: row[:name],
      api_id: row[:id],
      ticker: row[:symbol].upcase,
      icon: row[:image],
      rate: row[:current_price],
      price_change: row[:price_change_percentage_24h]
    }
  end
  private_class_method :coin_params
end
