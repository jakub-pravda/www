function populateTable(pricingSection) {
    // Fetch the JSON data
    fetch('assets/json/pricing.json')
      .then(response => response.json())
      .then(data => {
        // Get the table body
        const tbody = document.querySelector(`.table.${pricingSection} tbody`);
  
        // Clear the table body
        tbody.innerHTML = '';
  
        // Get the data for the specified pricing section
        const sectionData = data[pricingSection];
  
        // Iterate over the section data
        for (const key in sectionData) {
          const item = sectionData[key];
  
          // Create a new row and cells
          const row = document.createElement('tr');
          const nameCell = document.createElement('td');
          const priceCell = document.createElement('td');
  
          // Set the cell text
          nameCell.textContent =  item.name.replace('m3', 'm³');
          priceCell.textContent = item.price.replace('m3', 'm³');
  
          // CSS styling of table cells
          nameCell.style.fontWeight = '500';
          priceCell.style.textAlign = 'right';

          // Add the cells to the row
          row.appendChild(nameCell);
          row.appendChild(priceCell);
  
          // Add the row to the table
          tbody.appendChild(row);
        }
      });
  }