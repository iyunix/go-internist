// JS for landing page buttons

document.addEventListener('DOMContentLoaded', function() {
  // Connect Get Started button to login page
  var getStartedBtn = document.querySelector('.landing-btn');
  if (getStartedBtn) {
    getStartedBtn.addEventListener('click', function(e) {
      e.preventDefault();
      window.location.href = '/login';
    });
  }

  // Connect Sign Up button to register page
  var signUpBtn = document.querySelector('.cta-container .btn');
  if (signUpBtn) {
    signUpBtn.addEventListener('click', function(e) {
      e.preventDefault();
      window.location.href = '/register';
    });
  }
});
