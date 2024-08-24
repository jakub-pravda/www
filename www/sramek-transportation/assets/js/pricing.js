fetch('assets/json/pricing.json')
  .then(response => response.json())
  .then(data => {
    for (let section in data) {
      for (let key in data[section]) {
        let name = document.querySelector(`#${section}-${key}-name`);
        let price = document.querySelector(`#${section}-${key}-price`);
        
        if (name && price) {
          name.textContent = data[section][key].name;
          price.textContent = data[section][key].price;
        }
      }
    }
  })
  .catch(error => console.error('Error:', error));