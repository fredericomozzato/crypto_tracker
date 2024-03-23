class CreateHoldings < ActiveRecord::Migration[7.1]
  def change
    create_table :holdings do |t|
      t.references :portfolio, null: false, foreign_key: true
      t.references :coin, null: false, foreign_key: true
      t.decimal :amount, precision: 16, scale: 8, default: 0.0

      t.timestamps

      t.index [:portfolio_id, :coin_id], unique: true, name: 'portfolio_coin_id'
    end
  end
end
