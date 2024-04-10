module ApplicationHelper
  def number_format(number)
    number_to_currency(
      number_to_human(
        number,
        units: { thousand: 'K', million: 'M', billion: 'B' },
        format: '%n%u',
        precision: 2,
        significant: false
      )
    )
  end
end
