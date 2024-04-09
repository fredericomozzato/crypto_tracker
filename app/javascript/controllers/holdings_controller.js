import { Controller } from "@hotwired/stimulus"

// Connects to data-controller="holdings"
export default class extends Controller {
  static targets = ["priceChange"];
  static values  = {
    delta: Number
  }
  
  connect() {
  }
  
  priceChangeTargetConnected() {    
    if (this.deltaValue > 0) {
      this.priceChangeTarget.className += " text-green-700";
    } else if (this.deltaValue < 0) {
      this.priceChangeTarget.className += " text-red-700";
    } else {
      this.priceChangeTarget.className += " text-slate-500";
    }
  }
}
