// G:\go_internist\web\static\js\auth.js
function isValidIranianPhone(phone) {
    // Strict Iranian: starts with 09, 11 digits
    return /^09\d{9}$/.test(phone);
}

function validateRegistrationForm() {
    if (!validatePassword()) return false;

    const phoneInput = document.getElementById('phone-number');
    if (!isValidIranianPhone(phoneInput.value.trim())) {
        alert("شماره موبایل باید مانند 09123456789 باشد.");
        phoneInput.focus();
        return false;
    }
    return true;
}
window.validateRegistrationForm = validateRegistrationForm;

