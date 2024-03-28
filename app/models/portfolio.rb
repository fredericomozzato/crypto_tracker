class Portfolio < ApplicationRecord
  belongs_to :account
  has_many :holdings, dependent: :destroy

  delegate :owner, to: :account

  validates :name, presence: true

  def total_balance
    holdings.map(&:value).sum
  end
end
