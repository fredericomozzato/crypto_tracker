class AddUniqueIndexToAccountOwner < ActiveRecord::Migration[7.1]
  def change
    add_index :accounts, :owner_id, unique: true, name: 'unique_account_owner'
  end
end
