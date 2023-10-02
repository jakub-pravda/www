const placeId = 'ChIJefayK7LxC0cRM64ybg9kg7Q'; // The Google Place ID of the shop

// Get the current day of the week (0 = Sunday, 1 = Monday, ..., 6 = Saturday)
const currentDay = new Date().getDay();
console.log(`Today is ${currentDay}.`);

var request = {
  placeId: placeId,
  fields: ['name', 'opening_hours']
};

service = new google.maps.places.PlacesService(document.createElement('div'));
service.getDetails(request, callback)
.catch(error => console.error('Error fetching data:', error));;

function callback(place, status) {
  if (status == google.maps.places.PlacesServiceStatus.OK) {
    console.log(place);

    const openingHours = place.opening_hours;

    if (openingHours) {
      const todayOpeningHours = openingHours.periods
      .filter(period => period.open.day === currentDay)
      .map(openingHours => `${convertTime(openingHours.open.time)} - ${convertTime(openingHours.close.time)}`)
      .join(', ');

      console.log(todayOpeningHours);
      resultElement = document.getElementById('openinghours')
      if (todayOpeningHours.length > 0) {
        const text = document.createTextNode(`DNES OTEVŘENO: ${todayOpeningHours}`);
        resultElement.appendChild(text);
      } else {
        const text = document.createTextNode(`DNES MÁME ZAVŘENO`);
        resultElement.appendChild(text);
      }
    } else {
      console.log('Unable to retrieve opening hours.');
    }
  }
}

function convertTime(time) {
  const hours = time.substring(0, 2);
  const minutes = time.substring(2);
  return `${hours}:${minutes}`;
}
