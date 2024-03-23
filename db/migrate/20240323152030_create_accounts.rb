class CreateAccounts < ActiveRecord::Migration[7.1]
  def change
    create_table :accounts do |t|
      t.references :owner, null: false, foreign_key: { to_table: :users }
      t.string :uuid, null: false, index: { unique: true, name: 'unique_account_uuid' }

      t.timestamps
    end
  end
end
