class CoinService
  def self.import_coins
    res = JSON.parse GeckoService.top_markets, symbolize_names: true

    res.each do |r|
      Coin.create name: r[:name], api_id: r[:id], ticker: r[:symbol],
                  icon: r[:image], rate: r[:current_price], active: true
    end
  end
end
