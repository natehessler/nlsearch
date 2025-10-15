const queryInput = document.getElementById('queryInput');
const searchBtn = document.getElementById('searchBtn');
const clearBtn = document.getElementById('clearBtn');
const loadingDiv = document.getElementById('loadingDiv');
const resultDiv = document.getElementById('resultDiv');
const examples = document.querySelectorAll('.example-item');

async function performSearch() {
    const query = queryInput.value.trim();
    if (!query) return;

    searchBtn.disabled = true;
    loadingDiv.classList.remove('hidden');
    resultDiv.classList.add('hidden');

    try {
        const response = await fetch('/api/query', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ query }),
        });

        const data = await response.json();

        if (data.error) {
            showError(data.error);
        } else {
            showResult(data);
        }
    } catch (error) {
        showError('Network error: ' + error.message);
    } finally {
        searchBtn.disabled = false;
        loadingDiv.classList.add('hidden');
    }
}

function showResult(data) {
    let html = '<div class="result">';
    html += '<h3>Generated Search Query</h3>';
    html += `<div class="answer"><code>${escapeHtml(data.answer)}</code></div>`;
    html += '</div>';
    resultDiv.innerHTML = html;
    resultDiv.classList.remove('hidden');
}

function showError(message) {
    resultDiv.innerHTML = `<div class="error">❌ ${escapeHtml(message)}</div>`;
    resultDiv.classList.remove('hidden');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

searchBtn.addEventListener('click', performSearch);

clearBtn.addEventListener('click', () => {
    queryInput.value = '';
    resultDiv.classList.add('hidden');
    queryInput.focus();
});

queryInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        performSearch();
    }
});

examples.forEach(example => {
    example.addEventListener('click', () => {
        queryInput.value = example.dataset.query;
        queryInput.focus();
    });
});
