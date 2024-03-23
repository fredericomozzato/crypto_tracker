class Account < ApplicationRecord
  belongs_to :owner, class_name: 'User'
  has_many :portfolios, dependent: :destroy
  has_many :holdings, through: :portfolio

  validates :uuid, presence: true
  validates :owner, :uuid, uniqueness: true

  before_validation :generate_uuid, on: :create

  private

  def generate_uuid
    self.uuid = SecureRandom.uuid
  end
end
