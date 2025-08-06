const tg = window.Telegram.WebApp;

// Initialize Telegram Web App
if (tg) {
    tg.ready();
    document.body.classList.add('tg-app');
}

// Calendar functionality
let currentDate = new Date();
let selectedDate = new Date();

function initCalendar() {
    updateCalendar();
    
    document.getElementById('prev-month').addEventListener('click', () => {
        currentDate.setMonth(currentDate.getMonth() - 1);
        updateCalendar();
    });
    
    document.getElementById('next-month').addEventListener('click', () => {
        currentDate.setMonth(currentDate.getMonth() + 1);
        updateCalendar();
    });
}

function updateCalendar() {
    const monthNames = [
        'January', 'February', 'March', 'April', 'May', 'June',
        'July', 'August', 'September', 'October', 'November', 'December'
    ];
    
    document.getElementById('current-month').textContent = 
        `${monthNames[currentDate.getMonth()]} ${currentDate.getFullYear()}`;
    
    const grid = document.getElementById('calendar-grid');
    grid.innerHTML = '';
    
    // Add day headers
    const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    days.forEach(day => {
        const dayHeader = document.createElement('div');
        dayHeader.className = 'calendar-day';
        dayHeader.textContent = day;
        dayHeader.style.fontWeight = 'bold';
        dayHeader.style.color = '#667eea';
        grid.appendChild(dayHeader);
    });
    
    // Get first day of month and number of days
    const firstDay = new Date(currentDate.getFullYear(), currentDate.getMonth(), 1);
    const lastDay = new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 0);
    const startDate = new Date(firstDay);
    startDate.setDate(startDate.getDate() - firstDay.getDay());
    
    // Fill calendar
    for (let i = 0; i < 42; i++) {
        const date = new Date(startDate);
        date.setDate(startDate.getDate() + i);
        
        const dayElement = document.createElement('div');
        dayElement.className = 'calendar-day';
        dayElement.textContent = date.getDate();
        
        // Check if it's today
        const today = new Date();
        if (date.toDateString() === today.toDateString()) {
            dayElement.classList.add('today');
        }
        
        // Check if it's selected
        if (date.toDateString() === selectedDate.toDateString()) {
            dayElement.classList.add('selected');
        }
        
        // Check if it's from other month
        if (date.getMonth() !== currentDate.getMonth()) {
            dayElement.classList.add('other-month');
        }
        
        // Add click handler
        dayElement.addEventListener('click', () => {
            selectedDate = date;
            updateCalendar();
            updateDateInput();
        });
        
        grid.appendChild(dayElement);
    }
}

function updateDateInput() {
    const dateInput = document.getElementById('date');
    const year = selectedDate.getFullYear();
    const month = String(selectedDate.getMonth() + 1).padStart(2, '0');
    const day = String(selectedDate.getDate()).padStart(2, '0');
    dateInput.value = `${year}-${month}-${day}`;
}

// Form handling
document.getElementById('expense-form').addEventListener('submit', function(e) {
    e.preventDefault();
    
    const formData = new FormData(this);
    const data = {
        date: formData.get('date'),
        category: formData.get('category'),
        description: formData.get('description'),
        amount: parseFloat(formData.get('amount'))
    };
    
    // Validate data
    if (!data.date || !data.category || !data.amount || data.amount <= 0) {
        showMessage('Please fill in all required fields correctly.', 'error');
        return;
    }
    
    // Send to server
    fetch('/transaction', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(data)
    })
    .then(response => response.json())
    .then(result => {
        if (result.error) {
            showMessage(result.error, 'error');
        } else {
            showMessage('✅ Expense added successfully!', 'success');
            this.reset();
            updateDateInput(); // Reset to selected date
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showMessage('❌ Failed to add expense. Please try again.', 'error');
    });
    
    // Send to Telegram if available
    if (tg) {
        tg.sendData(JSON.stringify(data));
    }
});

// CSV upload handling
document.getElementById('csv-form').addEventListener('submit', function(e) {
    e.preventDefault();
    
    const formData = new FormData(this);
    const file = formData.get('csv');
    
    if (!file) {
        showMessage('Please select a CSV file.', 'error');
        return;
    }
    
    // Show loading state
    const submitBtn = this.querySelector('.upload-btn');
    const originalText = submitBtn.textContent;
    submitBtn.textContent = '📤 Uploading...';
    submitBtn.disabled = true;
    
    fetch('/upload-csv', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(result => {
        if (result.error) {
            showMessage(result.error, 'error');
            if (result.errors) {
                result.errors.forEach(error => {
                    showMessage(error, 'error');
                });
            }
        } else {
            showMessage(`✅ ${result.message}`, 'success');
            this.reset();
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showMessage('❌ Failed to upload CSV. Please try again.', 'error');
    })
    .finally(() => {
        submitBtn.textContent = originalText;
        submitBtn.disabled = false;
    });
});

function showMessage(message, type) {
    // Remove existing messages
    const existingMessages = document.querySelectorAll('.message');
    existingMessages.forEach(msg => msg.remove());
    
    // Create new message
    const messageElement = document.createElement('div');
    messageElement.className = `message ${type}`;
    messageElement.textContent = message;
    
    // Insert at the top of the container
    const container = document.querySelector('.container');
    container.insertBefore(messageElement, container.firstChild);
    
    // Auto-remove after 5 seconds
    setTimeout(() => {
        messageElement.remove();
    }, 5000);
}

// Initialize everything when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    initCalendar();
    updateDateInput();
    
    // Set today's date as default
    selectedDate = new Date();
    updateCalendar();
    updateDateInput();
});
