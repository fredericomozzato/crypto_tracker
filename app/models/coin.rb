class Coin < ApplicationRecord
  has_many :holdings, dependent: :destroy

  validates :name, :api_id, :ticker, :icon, :active, presence: true
  validates :name, :api_id, :ticker, :icon, uniqueness: true
  validates :rate, numericality: { greater_than_or_equal_to: 0 }
end
