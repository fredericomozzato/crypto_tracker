class Holding < ApplicationRecord
  belongs_to :portfolio
  belongs_to :coin

  delegate :ticker, :rate, :price_change, to: :coin

  validates :amount, numericality: { greater_than_or_equal_to: 0 }
  validates :portfolio, uniqueness: { scope: :coin }

  def value
    amount * coin.rate
  end

  def deposit(amount)
    self.amount += amount if amount.positive?
  end

  def withdraw(amount)
    self.amount -= amount if amount.positive? && amount <= self.amount
  end

  def proportion
    return 0 if portfolio.total_balance.zero?

    (value / portfolio.total_balance * 100).round 2
  end
end
