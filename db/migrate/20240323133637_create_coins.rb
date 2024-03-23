class CreateCoins < ActiveRecord::Migration[7.1]
  def change
    create_table :coins do |t|
      t.string :name, null: false
      t.string :api_id, null: false
      t.string :ticker, null: false
      t.string :icon, null: false
      t.decimal :rate, precision: 16, scale: 8, default: 0.0

      t.timestamps
    end
  end
end
