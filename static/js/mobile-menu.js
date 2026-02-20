document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('#mobile-nav a').forEach(a => {
        a.addEventListener('click', () => {
            document.getElementById('mobile-menu-toggle').checked = false;
        });
    });
});