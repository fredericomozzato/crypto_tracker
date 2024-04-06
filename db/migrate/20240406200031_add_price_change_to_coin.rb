class AddPriceChangeToCoin < ActiveRecord::Migration[7.1]
  def change
    add_column :coins, :price_change, :float, null: false
  end
end
