// G:\go_internist\web\static\js\markdown_renderer.js
// Fixed markdown renderer - corrected regex syntax

class MarkdownRenderer {
    constructor() {
        this.marked = window.marked;
        this.DOMPurify = window.DOMPurify;
        
        if (!this.marked) {
            console.error('Marked.js not loaded - markdown rendering disabled');
            return;
        }
        
        if (!this.DOMPurify) {
            console.warn('DOMPurify not loaded - HTML sanitization disabled');
        }

        this.setupMarked();
        console.log('✅ MarkdownRenderer initialized');
    }

    setupMarked() {
        if (!this.marked) return;

        // Configure marked for medical content
        this.marked.setOptions({
            breaks: true,
            gfm: true,
            headerIds: false,
            mangle: false,
            sanitize: false
        });

        // Custom renderer for medical content
        const renderer = new this.marked.Renderer();
        
        // Enhanced code blocks
        renderer.code = (code, language) => {
            const lang = language ? ` class="language-${language}"` : '';
            return `<pre class="medical-code"><code${lang}>${this.escapeHtml(code)}</code></pre>`;
        };

        // Enhanced lists
        renderer.list = (body, ordered, start) => {
            const type = ordered ? 'ol' : 'ul';
            const startAttr = ordered && start !== 1 ? ` start="${start}"` : '';
            return `<${type} class="medical-list"${startAttr}>\n${body}</${type}>\n`;
        };

        // Enhanced emphasis
        renderer.strong = (text) => `<strong class="medical-emphasis">${text}</strong>`;

        this.marked.use({ renderer });
    }

    // ✅ REAL-TIME: Render markdown progressively during streaming
    renderStreaming(content, container) {
        if (!this.marked || !content || !container) {
            if (container) container.textContent = content || '';
            return;
        }

        try {
            // ✅ FIXED: Handle partial markdown during streaming
            const processedContent = this.preprocessStreamingContent(content);
            const html = this.marked.parse(processedContent);
            const sanitized = this.DOMPurify ? this.DOMPurify.sanitize(html) : html;
            
            container.innerHTML = sanitized;

            // ✅ Enhance medical content
            this.enhanceMedicalContent(container);
            
            // ✅ Smooth scroll to bottom
            this.scrollToBottom(container);

        } catch (error) {
            console.warn('Markdown rendering error during streaming:', error);
            container.textContent = content;
        }
    }

    // ✅ COMPLETE: Render final markdown after streaming completes
    renderComplete(content, container) {
        if (!this.marked || !content || !container) {
            if (container) container.textContent = content || '';
            return;
        }

        try {
            const html = this.marked.parse(content);
            const sanitized = this.DOMPurify ? this.DOMPurify.sanitize(html) : html;
            
            container.innerHTML = sanitized;

            // ✅ Final enhancements
            this.enhanceMedicalContent(container);
            this.addCopyButtons(container);
            
        } catch (error) {
            console.error('Final markdown rendering error:', error);
            container.textContent = content;
        }
    }

    // ✅ PREPROCESSING: Handle incomplete markdown during streaming
    preprocessStreamingContent(content) {
        if (!content) return '';

        // Handle incomplete code blocks
        const codeBlockCount = (content.match(/```
        if (codeBlockCount % 2 === 1) {
            content += '\n```';
        }

        // Handle incomplete lists
        const lines = content.split('\n');
        const lastLine = lines[lines.length - 1];
        
        // ✅ FIXED REGEX: Properly escaped regex pattern
        if (/^[\s]*[-*+]\s*$/.test(lastLine)) {
            content += ' ';
        }

        return content;
    }

    // ✅ MEDICAL: Enhance medical terminology and formatting
    enhanceMedicalContent(container) {
        if (!container) return;

        try {
            // Highlight medical terms
            const medicalTerms = [
                'hypertension', 'diabetes', 'cardiovascular', 'myocardial',
                'pulmonary', 'renal', 'hepatic', 'neurological', 'oncology',
                'pharmacology', 'dosage', 'contraindication', 'side effect',
                'diagnosis', 'prognosis', 'treatment', 'therapy', 'medication'
            ];

            medicalTerms.forEach(term => {
                // ✅ FIXED: Properly constructed regex with escaped backslashes
                const regex = new RegExp('\\b(' + term + ')\\b', 'gi');
                container.innerHTML = container.innerHTML.replace(regex, 
                    '<span class="medical-term">$1</span>');
            });

            // Highlight drug names
            // ✅ FIXED: Properly escaped regex pattern
            const drugRegex = /\b([A-Z][a-z]*(?:ol|ine|ate|ide|cin))\b/g;
            container.innerHTML = container.innerHTML.replace(drugRegex,
                '<span class="drug-name">$1</span>'
            );

            // Add icons to medical lists
            const lists = container.querySelectorAll('ul.medical-list');
            lists.forEach(list => {
                list.querySelectorAll('li').forEach(item => {
                    const text = item.textContent.toLowerCase();
                    if (text.includes('symptom')) {
                        item.classList.add('symptom-item');
                    } else if (text.includes('treatment')) {
                        item.classList.add('treatment-item');
                    }
                });
            });

        } catch (error) {
            console.warn('Medical content enhancement error:', error);
        }
    }

    // ✅ UTILITY: Add copy buttons to code blocks
    addCopyButtons(container) {
        if (!container) return;

        try {
            const codeBlocks = container.querySelectorAll('pre.medical-code');
            codeBlocks.forEach(block => {
                if (block.querySelector('.copy-btn')) return;

                const copyBtn = document.createElement('button');
                copyBtn.className = 'copy-btn';
                copyBtn.innerHTML = '📋 Copy';
                copyBtn.onclick = () => this.copyToClipboard(block.textContent, copyBtn);
                
                block.style.position = 'relative';
                block.appendChild(copyBtn);
            });
        } catch (error) {
            console.warn('Copy button error:', error);
        }
    }

    // ✅ UTILITY: Copy text to clipboard
    async copyToClipboard(text, button) {
        try {
            await navigator.clipboard.writeText(text);
            const originalText = button.innerHTML;
            button.innerHTML = '✅ Copied!';
            button.style.background = '#4CAF50';
            
            setTimeout(() => {
                button.innerHTML = originalText;
                button.style.background = '';
            }, 2000);
        } catch (error) {
            console.error('Copy failed:', error);
            button.innerHTML = '❌ Failed';
        }
    }

    // ✅ UTILITY: Smooth scroll to bottom
    scrollToBottom(container) {
        try {
            const messagesContainer = container.closest('.messages-container') || 
                                    container.closest('.messages') ||
                                    container.closest('#messages-container');
            if (messagesContainer) {
                messagesContainer.scrollTop = messagesContainer.scrollHeight;
            }
        } catch (error) {
            console.warn('Scroll error:', error);
        }
    }

    // ✅ UTILITY: Escape HTML
    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// ✅ Export singleton instance
const markdownRenderer = new MarkdownRenderer();
window.MarkdownRenderer = markdownRenderer;

console.log('✅ MarkdownRenderer loaded and ready');
