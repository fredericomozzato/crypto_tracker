class Holding < ApplicationRecord
  belongs_to :portfolio
  belongs_to :coin

  delegate :ticker, :rate, to: :coin

  validates :amount, numericality: { greater_than_or_equal_to: 0 }
  validates :portfolio, uniqueness: { scope: :coin }

  def value
    amount * coin.rate
  end

  def deposit(amount)
    self.amount += amount if amount.positive?
    save
  end

  def withdraw(amount)
    self.amount -= amount if amount.positive? && amount <= self.amount
    save
  end
end
