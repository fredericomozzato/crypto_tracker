class AddUniquenessIndexToCoin < ActiveRecord::Migration[7.1]
  def change
    add_index :coins, :name, unique: true, name: 'unique_coin_name'
    add_index :coins, :api_id, unique: true, name: 'unique_coin_api_id'
    add_index :coins, :ticker, unique: true, name: 'unique_coin_ticker'
    add_index :coins, :icon, unique: true, name: 'unique_coin_icon'
  end
end
