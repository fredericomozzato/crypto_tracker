class Account < ApplicationRecord
  belongs_to :owner, class_name: 'User'
  has_many :portfolios, dependent: :destroy
  has_many :holdings, through: :portfolios

  validates :uuid, presence: true
  validates :owner, :uuid, uniqueness: true

  before_validation :generate_uuid, on: :create

  def net_worth
    portfolios.map(&:total_balance).sum
  end

  def assets
    holdings.group(:coin).sum(:amount)
  end

  private

  def generate_uuid
    self.uuid = SecureRandom.uuid
  end
end
