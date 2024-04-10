module ApplicationHelper
  def balance_format(balance)
    number_to_currency(
      number_to_human(
        balance,
        units: { thousand: 'K', million: 'M', billion: 'B' },
        format: '%n%u',
        precision: 2,
        significant: false
      )
    )
  end

  def asset_percentage(coin, amount, net_worth)
    number_to_percentage coin.rate * amount / net_worth * 100,
                         precision: 2
  end
end
