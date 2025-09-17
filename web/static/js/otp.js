// G:\go_internist\web\static\js\otp.js
const otpContainer = document.getElementById('otp-container');
const inputs = [...otpContainer.querySelectorAll('.otp-input')];
const hiddenInput = document.getElementById('sms_code_hidden');

inputs.forEach((input, index) => {
    input.addEventListener('input', (e) => {
        // Only allow numeric digits
        e.target.value = e.target.value.replace(/\D/g, '');

        if (e.target.value.length === 1 && index < inputs.length - 1) {
            inputs[index + 1].focus();
        }
        updateHiddenInput();
    });

    
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Backspace' && !e.target.value && index > 0) {
            inputs[index - 1].focus();
        }
    });
    
    input.addEventListener('paste', (e) => {
        e.preventDefault();
        let paste = (e.clipboardData || window.clipboardData).getData('text').replace(/\D/g, '').slice(0, 6);
        paste.split('').forEach((char, i) => {
            if (inputs[i]) { inputs[i].value = char; }
        });
        updateHiddenInput();
        const lastInput = inputs[paste.length - 1] || inputs[inputs.length - 1];
        lastInput.focus();
    });
});

function updateHiddenInput() {
    const code = inputs.map(input => input.value).join('');
    hiddenInput.value = code;
}
