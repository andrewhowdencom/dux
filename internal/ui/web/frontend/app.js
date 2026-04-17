const md = window.markdownit();

const elements = {
    chatForm: document.getElementById('chatForm'),
    promptInput: document.getElementById('promptInput'),
    sendBtn: document.getElementById('sendBtn'),
    chatContainer: document.getElementById('chatContainer')
};

// State
let isGenerating = false;

// Initialize
async function initChat() {
    try {
        // Provision the session cookie
        await fetch('/api/session', { method: 'POST' });
        
        elements.sendBtn.disabled = false;
    } catch (e) {
        console.error("Failed to initialize session", e);
    }
}

// Auto-resize textarea
elements.promptInput.addEventListener('input', function() {
    this.style.height = 'auto';
    this.style.height = (this.scrollHeight) + 'px';
});

elements.promptInput.addEventListener('keydown', function(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        elements.chatForm.dispatchEvent(new Event('submit'));
    }
});

// Create Message Bubble
function createMessageBubble(role) {
    const wrap = document.createElement('div');
    wrap.className = `message ${role}`;
    
    const content = document.createElement('div');
    content.className = 'message-content';
    wrap.appendChild(content);
    
    elements.chatContainer.appendChild(wrap);
    elements.chatContainer.scrollTop = elements.chatContainer.scrollHeight;
    
    return content;
}

// Handle Form Submit
elements.chatForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (isGenerating || !elements.promptInput.value.trim()) return;
    
    const prompt = elements.promptInput.value.trim();
    
    // Add user message
    const userMessageContent = createMessageBubble('user');
    userMessageContent.innerHTML = md.render(prompt);
    
    // Reset input
    elements.promptInput.value = '';
    elements.promptInput.style.height = 'auto';
    
    await generateResponse(prompt);
});

async function resolveHitl(callId, approve, hitlElement) {
    hitlElement.querySelectorAll('button').forEach(b => b.disabled = true);
    try {
        const res = await fetch('/api/chat/approve', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({call_id: callId, approve})
        });
        if (res.ok) {
            hitlElement.style.opacity = '0.5';
            hitlElement.querySelector('.hitl-header').innerHTML = approve ? '✅ Tool Approved' : '❌ Tool Denied';
        } else {
            alert('Failed to resolve tool request');
        }
    } catch(e) {
        console.error(e);
    }
}

function createHitlBlock(contentEl, data) {
    const block = document.createElement('div');
    block.className = 'hitl-block';
    block.dataset.callId = data.call_id;
    
    const header = document.createElement('div');
    header.className = 'hitl-header';
    const icon = data.icon || '🔧';
    header.innerHTML = `${icon} <strong>Approval Required:</strong> ${data.tool}`;
    
    if (data.show_args !== false && data.args) {
        const body = document.createElement('div');
        body.className = 'hitl-body';
        body.textContent = JSON.stringify(data.args, null, 2);
        block.appendChild(body);
    }
    
    const actions = document.createElement('div');
    actions.className = 'hitl-actions';
    
    const denyBtn = document.createElement('button');
    denyBtn.className = 'btn-danger';
    denyBtn.textContent = 'Deny';
    denyBtn.onclick = () => resolveHitl(data.call_id, false, block);
    
    const allowBtn = document.createElement('button');
    allowBtn.className = 'btn-success';
    allowBtn.textContent = 'Approve';
    allowBtn.onclick = () => resolveHitl(data.call_id, true, block);
    
    actions.appendChild(denyBtn);
    actions.appendChild(allowBtn);
    
    block.appendChild(header);
    block.appendChild(actions);
    
    contentEl.appendChild(block);
    elements.chatContainer.scrollTop = elements.chatContainer.scrollHeight;
}

function createToolCallBlock(contentEl, data) {
    const block = document.createElement('div');
    block.className = 'tool-call-block';
    block.dataset.tool = data.tool;
    
    const header = document.createElement('div');
    header.className = 'tool-call-header';
    const icon = data.icon || '🔧';
    header.innerHTML = `${icon} <strong>Tool:</strong> ${data.tool}`;
    
    if (data.show_args !== false && data.args) {
        const body = document.createElement('div');
        body.className = 'tool-call-body';
        body.textContent = JSON.stringify(data.args, null, 2);
        block.appendChild(body);
    }
    
    const status = document.createElement('div');
    status.className = 'tool-call-status';
    status.innerHTML = '⏳ <em>Executing...</em>';
    block.appendChild(status);
    
    contentEl.appendChild(block);
    elements.chatContainer.scrollTop = elements.chatContainer.scrollHeight;
}

function createToolResultBlock(contentEl, data) {
    const block = document.createElement('div');
    block.className = 'tool-result-block';
    
    const header = document.createElement('div');
    header.className = 'tool-result-header';
    const icon = data.icon || '🔧';
    const statusIcon = data.is_error ? '❌' : '✅';
    header.innerHTML = `${icon} <strong>Result:</strong> ${data.tool} ${statusIcon}`;
    
    if (data.show_result !== false && data.result) {
        const body = document.createElement('div');
        body.className = 'tool-result-body';
        
        let resText = data.result;
        if (resText.length > 500) {
            resText = resText.substring(0, 500) + ' ... (truncated)';
        }
        
        if (data.is_error) {
            body.style.color = 'var(--danger)';
            resText = "Error: " + resText;
        }
        
        body.textContent = resText;
        block.appendChild(body);
    }
    
    contentEl.appendChild(block);
    elements.chatContainer.scrollTop = elements.chatContainer.scrollHeight;
}

function createTelemetryFooter(contentEl, data) {
    const footer = document.createElement('div');
    footer.className = 'telemetry-footer';
    
    let resStr = "";
    if (data.reasoning_tokens && data.reasoning_tokens > 0) {
        resStr = ` (including ${data.reasoning_tokens} reasoning)`;
    }
    
    footer.textContent = `⚡ ${data.duration_secs.toFixed(1)}s | Tokens: ${data.input_tokens} in, ${data.output_tokens} out${resStr}`;
    contentEl.appendChild(footer);
    elements.chatContainer.scrollTop = elements.chatContainer.scrollHeight;
}

// Parse NDJSON Stream
async function generateResponse(prompt) {
    isGenerating = true;
    elements.sendBtn.disabled = true;
    elements.promptInput.disabled = true;
    
    const assistantContent = createMessageBubble('assistant');
    let currentTextEl = null;
    let currentFullText = '';
    let currentThinkingEl = null;

    function getTextEl() {
        if (!currentTextEl) {
            currentTextEl = document.createElement('div');
            assistantContent.appendChild(currentTextEl);
            currentFullText = '';
        }
        return currentTextEl;
    }

    function getThinkingEl() {
        if (!currentThinkingEl) {
            currentThinkingEl = document.createElement('div');
            currentThinkingEl.className = 'thinking-text';
            currentThinkingEl.textContent = 'Thinking... ';
            assistantContent.appendChild(currentThinkingEl);
        }
        return currentThinkingEl;
    }
    
    try {
        const response = await fetch('/api/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({prompt})
        });
        
        if (!response.ok) {
            const errText = await response.text();
            throw new Error(errText || `HTTP ${response.status}`);
        }
        
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        
        while (true) {
            const {value, done} = await reader.read();
            if (done) break;
            
            buffer += decoder.decode(value, {stream: true});
            const lines = buffer.split('\n');
            buffer = lines.pop(); // Keep incomplete line in buffer
            
            for (const line of lines) {
                if (!line.trim()) continue;
                try {
                    const data = JSON.parse(line);
                    if (data.type === 'text') {
                        currentFullText += data.content;
                        getTextEl().innerHTML = md.render(currentFullText);
                    } else if (data.type === 'thinking') {
                        getThinkingEl().textContent += data.content;
                    } else if (data.type === 'hitl_request') {
                        createHitlBlock(assistantContent, data);
                        currentTextEl = null;
                    } else if (data.type === 'tool_call') {
                        if (currentThinkingEl) {
                            currentThinkingEl.style.display = 'none';
                        }
                        createToolCallBlock(assistantContent, data);
                        currentTextEl = null;
                    } else if (data.type === 'tool_result') {
                        const toolBlocks = assistantContent.querySelectorAll('.tool-call-block');
                        toolBlocks.forEach(block => {
                            if (block.dataset.tool === data.tool && !block.classList.contains('completed')) {
                                block.classList.add('completed');
                                const statusEl = block.querySelector('.tool-call-status');
                                if (statusEl) {
                                    statusEl.innerHTML = data.is_error ? '❌ <em>Failed</em>' : '✅ <em>Completed</em>';
                                }
                            }
                        });
                        createToolResultBlock(assistantContent, data);
                        currentTextEl = null;
                    } else if (data.type === 'telemetry') {
                        createTelemetryFooter(assistantContent, data);
                    } else if (data.type === 'error') {
                        getTextEl().innerHTML += `<br><span style="color:var(--danger)">${data.error}</span>`;
                    } else if (data.type === 'command') {
                        if (data.command === '/new') {
                             fetch('/api/session', { method: 'POST' }).catch(console.error);
                             elements.chatContainer.innerHTML = '';
                             elements.chatContainer.appendChild(assistantContent);
                             getTextEl().innerHTML = `<em>Started a new conversation session.</em>`;
                        }
                    }
                    elements.chatContainer.scrollTop = elements.chatContainer.scrollHeight;
                } catch(e) {
                    console.error('Error parsing line:', line, e);
                }
            }
        }
    } catch(e) {
        console.error(e);
        assistantContent.innerHTML += `<br><span style="color:var(--danger)">Connection Error: ${e.message}</span>`;
    } finally {
        isGenerating = false;
        elements.sendBtn.disabled = false;
        elements.promptInput.disabled = false;
        elements.promptInput.focus();
    }
}

// Start
initChat();
