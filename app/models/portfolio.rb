class Portfolio < ApplicationRecord
  belongs_to :account
  has_many :holdings, dependent: :destroy

  validates :name, presence: true
end
