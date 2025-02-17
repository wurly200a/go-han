// Global variable to store schedule data.
let scheduleData = null;
// Global language setting; default is Japanese.
let currentLang = "ja";

// Function to return weekday names based on current language.
function getWeekdayNames() {
  return currentLang === "ja"
    ? ["日", "月", "火", "水", "木", "金", "土"]
    : ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
}

// Cookie utility functions.
function setCookie(cname, cvalue, exdays) {
  const d = new Date();
  d.setTime(d.getTime() + (exdays * 24 * 60 * 60 * 1000));
  const expires = "expires=" + d.toUTCString();
  document.cookie = cname + "=" + cvalue + ";" + expires + ";path=/";
}
function getCookie(cname) {
  const name = cname + "=";
  const ca = document.cookie.split(';');
  for(let i = 0; i < ca.length; i++) {
    let c = ca[i].trim();
    if (c.indexOf(name) === 0) return c.substring(name.length, c.length);
  }
  return "";
}
// Retrieve language from cookie if available.
currentLang = getCookie("lang") || "ja";
$('#languageSelect').val(currentLang);

// Translation dictionary.
const translations = {
  en: {
    pageTitle: "Meal Schedule",
    startDateLabel: "Start Date (YYYY-MM-DD):",
    daysLabel: "Number of Days:",
    loadScheduleButton: "Load Schedule",
    dateHeader: "Date",
    lunchHeader: "Lunch",
    dinnerHeader: "Dinner"
  },
  ja: {
    pageTitle: "ごはん管理",
    startDateLabel: "開始日 (YYYY-MM-DD):",
    daysLabel: "日数:",
    loadScheduleButton: "スケジュール読み込み",
    dateHeader: "日付",
    lunchHeader: "昼",
    dinnerHeader: "夜"
  }
};

// Update UI texts based on current language.
function updateLanguage() {
  const t = translations[currentLang];
  $('#pageTitle').text(t.pageTitle);
  $('#startDateLabel').text(t.startDateLabel);
  $('#daysLabel').text(t.daysLabel);
  $('#loadSchedule').text(t.loadScheduleButton);
}

// Language dropdown change event.
$('#languageSelect').change(function() {
  currentLang = $(this).val();
  setCookie("lang", currentLang, 365);
  updateLanguage();
  if (scheduleData !== null) {
    renderSchedule(scheduleData);
  }
});

// Initial UI language update.
updateLanguage();

// Toggle options container.
$(document).ready(function() {
  $('#toggleOptions').click(function() {
    if ($('#optionsContainer').is(':visible')) {
      $('#optionsContainer').slideUp();
      $(this).text('▽');
    } else {
      $('#optionsContainer').slideDown();
      $(this).text('△');
    }
  });

  // Set default start date to today.
  function getToday() {
    const today = new Date();
    const yyyy = today.getFullYear();
    const mm = String(today.getMonth() + 1).padStart(2, '0');
    const dd = String(today.getDate()).padStart(2, '0');
    return `${yyyy}-${mm}-${dd}`;
  }
  $('#startDate').val(getToday());

  // Meal options mapping (numeric keys with text labels).
  const mealOptions = {
    1: "なし",
    2: "家",
    3: "弁当"
  };

  // Load schedule from backend.
  function loadSchedule() {
    const startDate = $('#startDate').val();
    const days = $('#days').val();
    $.ajax({
      url: '/api/meals',
      method: 'GET',
      data: { date: startDate, days: days },
      success: function(data) {
        scheduleData = data;
        renderSchedule(data);
      },
      error: function(err) {
        alert('Failed to load schedule.');
        console.error(err);
      }
    });
  }

  // Render schedule table.
  function renderSchedule(data) {
    $('#scheduleContainer').empty();
    const dates = Object.keys(data).sort();
    if (dates.length === 0) {
      $('#scheduleContainer').text('No schedule data.');
      return;
    }
    const weekdayNames = getWeekdayNames();
    // Extract user list from the first date's data.
    const firstDateMeals = data[dates[0]];
    const users = firstDateMeals.slice().sort((a, b) => a.user_id - b.user_id);

    let tableHtml = '<table>';
    // Header row 1: Date | each user (as link with colspan=2)
    tableHtml += '<thead><tr><th>' + translations[currentLang].dateHeader + '</th>';
    users.forEach(user => {
      tableHtml += '<th colspan="2"><a href="/user-defaults.html?user_id=' + user.user_id + '">' + user.user_name + '</a></th>';
    });
    tableHtml += '</tr>';
    // Header row 2: empty cell, then Lunch and Dinner for each user.
    tableHtml += '<tr><th></th>';
    users.forEach(() => {
      tableHtml += '<th>' + translations[currentLang].lunchHeader + '</th><th>' + translations[currentLang].dinnerHeader + '</th>';
    });
    tableHtml += '</tr></thead>';

    // Build table body.
    tableHtml += '<tbody>';
    dates.forEach(date => {
      tableHtml += '<tr>';
      const d = new Date(date);
      const weekday = d.getDay(); // 0: Sunday ... 6: Saturday
      const dayName = weekdayNames[weekday];
      // Format display as MM/DD (drop the year)
      const mm = String(d.getMonth() + 1).padStart(2, '0');
      const dd = String(d.getDate()).padStart(2, '0');
      const dateDisplay = mm + '/' + dd + ' (' + dayName + ')';
      let dateClass = "";
      if (weekday === 0) dateClass = "sunday";
      else if (weekday === 6) dateClass = "saturday";
      else dateClass = "weekday";
      tableHtml += '<td class="' + dateClass + '">' + dateDisplay + '</td>';

      // Create a mapping for this date.
      const mealMapping = {};
      data[date].forEach(meal => {
        mealMapping[meal.user_id] = meal;
      });

      users.forEach(user => {
        // The API returns numeric values.
        const meal = mealMapping[user.user_id] || { lunch: 0, dinner: 0, defaultLunch: 0, defaultDinner: 0 };
        // If no data (value 0), use default.
        const displayedLunch = meal.lunch === 0 ? meal.defaultLunch : meal.lunch;
        const displayedDinner = meal.dinner === 0 ? meal.defaultDinner : meal.dinner;
        // Determine cell classes.
        let lunchCellClass = "";
        let dinnerCellClass = "";
        if (meal.lunch === 0) {
          lunchCellClass = "cell-gray";
        } else if (meal.lunch !== meal.defaultLunch) {
          lunchCellClass = "cell-highlight";
        }
        if (meal.dinner === 0) {
          dinnerCellClass = "cell-gray";
        } else if (meal.dinner !== meal.defaultDinner) {
          dinnerCellClass = "cell-highlight";
        }
        // Build lunch dropdown.
        let lunchSelect = '<select class="lunchSelect" data-user-id="' + user.user_id + '" data-date="' + date + '">';
        for (let key in mealOptions) {
          lunchSelect += '<option value="' + key + '"';
          if (displayedLunch === parseInt(key)) {
            lunchSelect += ' selected';
          }
          lunchSelect += '>' + mealOptions[key] + '</option>';
        }
        lunchSelect += '</select>';
        // Build dinner dropdown.
        let dinnerSelect = '<select class="dinnerSelect" data-user-id="' + user.user_id + '" data-date="' + date + '">';
        for (let key in mealOptions) {
          dinnerSelect += '<option value="' + key + '"';
          if (displayedDinner === parseInt(key)) {
            dinnerSelect += ' selected';
          }
          dinnerSelect += '>' + mealOptions[key] + '</option>';
        }
        dinnerSelect += '</select>';
        tableHtml += '<td class="' + lunchCellClass + '">' + lunchSelect + '</td>';
        tableHtml += '<td class="' + dinnerCellClass + '">' + dinnerSelect + '</td>';
      });
      tableHtml += '</tr>';
    });
    tableHtml += '</tbody></table>';
    $('#scheduleContainer').html(tableHtml);
  }

  // Attach event handler so that when a dropdown changes, update that meal individually.
  $(document).on('change', 'select.lunchSelect, select.dinnerSelect', function() {
    const userId = $(this).attr('data-user-id');
    const date = $(this).attr('data-date');
    let payload = {
      user_id: parseInt(userId),
      date: date,
      lunch: 0,
      dinner: 0
    };
    if ($(this).hasClass("lunchSelect")) {
      const lunchVal = $(this).val();
      payload.lunch = parseInt(lunchVal);
    }
    if ($(this).hasClass("dinnerSelect")) {
      const dinnerVal = $(this).val();
      payload.dinner = parseInt(dinnerVal);
    }
    console.log('Updating meal for user ' + userId + ' on ' + date, payload);
    $.ajax({
      url: '/api/meals/bulk-update',
      method: 'PUT',
      contentType: 'application/json',
      data: JSON.stringify([payload]),
      success: function(response) {
        console.log('Meal updated for user ' + userId + ' on ' + date);
        loadSchedule();
      },
      error: function(err) {
        alert('Failed to update meal for user ' + userId);
        console.error(err);
      }
    });
  });

  // Load schedule on initial page load and when the "Load Schedule" button is clicked.
  loadSchedule();
  $('#loadSchedule').click(function() {
    loadSchedule();
  });
});
