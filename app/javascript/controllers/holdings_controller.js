import { Controller } from "@hotwired/stimulus"

// Connects to data-controller="holdings"
export default class extends Controller {
  static targets = ["priceChange"];
  static values  = {
    change: Number
  }
  
  connect() {
  }
  
  priceChangeTargetConnected() {
    if (this.changeValue > 0) {
      this.priceChangeTarget.className += " text-green-700";
    } else if (this.changeValue < 0) {
      this.priceChangeTarget.className += " text-red-700";
    } else {
      this.priceChangeTarget.className += " text-slate-500";
    }
  }
}
