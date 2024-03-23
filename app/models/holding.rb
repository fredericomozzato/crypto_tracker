class Holding < ApplicationRecord
  belongs_to :portfolio
  belongs_to :coin

  validates :amount, numericality: { greater_than_or_equal_to: 0 }
  validates :portfolio, uniqueness: { scope: :coin }

  def value
    amount * coin.rate
  end
end
