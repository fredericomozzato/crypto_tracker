module AssetsHelper
  def asset_value(coin, amount)
    number_to_currency coin.rate * amount
  end

  def asset_percentage(coin, amount, net_worth)
    number_to_percentage coin.rate * amount / net_worth * 100,
                         precision: 2
  end
end
