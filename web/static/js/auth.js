function validatePassword() {
    const password = document.getElementById('password').value;
    const confirmPasswordId = document.getElementById('confirm-password') ? 'confirm-password' : 'confirm_password';
    const confirmPassword = document.getElementById(confirmPasswordId).value;
    
    if (password !== confirmPassword) {
        alert("Passwords do not match.");
        return false;
    }
    return true;
}

// Make validatePassword available globally for onsubmit handlers
window.validatePassword = validatePassword;
