class User < ApplicationRecord
  has_one :account, foreign_key: 'owner_id', class_name: 'Account',
                    inverse_of: 'owner', dependent: :destroy

  devise :database_authenticatable, :registerable,
         :recoverable, :rememberable, :validatable
end
