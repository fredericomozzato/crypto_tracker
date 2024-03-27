class AddActiveToCoin < ActiveRecord::Migration[7.1]
  def change
    add_column :coins, :active, :boolean, null: false, default: true
  end
end
