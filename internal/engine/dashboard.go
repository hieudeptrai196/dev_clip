package engine

import (
	_ "embed"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
)

//go:embed appicon.png
var appIconBytes []byte

func (e *Engine) StartDashboard() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr := listener.Addr().(*net.TCPAddr)
	e.mu.Lock()
	e.dashboardPort = addr.Port
	e.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/", e.serveDashboardHTML)
	mux.HandleFunc("/api/stats", e.serveStats)
	mux.HandleFunc("/api/image", e.serveImage)
	mux.HandleFunc("/api/logo", e.serveLogo)

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			println("Dashboard server error:", err.Error())
		}
	}()
	return nil
}

func (e *Engine) DashboardPort() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dashboardPort
}

func (e *Engine) serveDashboardHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

func (e *Engine) serveLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "image/png")
	w.Write(appIconBytes)
}

func (e *Engine) serveImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	png := e.ItemImagePNG(id)
	if png == nil {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func (e *Engine) serveStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	e.mu.Lock()
	total := e.totalCopies
	counts := make(map[string]int)
	for k, v := range e.typeCounts {
		counts[k] = v
	}
	var copiesPerHour []int
	for _, val := range e.hourlyCopies {
		copiesPerHour = append(copiesPerHour, val)
	}
	e.mu.Unlock()

	history := e.cfg.Store.List()

	resp := map[string]interface{}{
		"total_copies":    total,
		"type_counts":     counts,
		"copies_per_hour": copiesPerHour,
		"history":         history,
	}
	json.NewEncoder(w).Encode(resp)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DevClip Dashboard</title>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        :root {
            --bg-color: #0b0c16;
            --card-bg: rgba(255, 255, 255, 0.03);
            --card-border: rgba(255, 255, 255, 0.08);
            --text-primary: #ffffff;
            --text-secondary: #8e92b2;
            --accent-glow: rgba(99, 102, 241, 0.15);
            --accent-color: #6366f1;
            --accent-success: #10b981;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Plus Jakarta Sans', sans-serif;
            background-color: var(--bg-color);
            color: var(--text-primary);
            min-height: 100vh;
            padding: 2rem;
            background-image: 
                radial-gradient(circle at 10% 20%, rgba(99, 102, 241, 0.05) 0%, transparent 40%),
                radial-gradient(circle at 90% 80%, rgba(236, 72, 153, 0.04) 0%, transparent 40%);
            background-attachment: fixed;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
        }

        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2.5rem;
            padding-bottom: 1.5rem;
            border-bottom: 1px solid var(--card-border);
        }

        .brand {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }

        .brand-logo {
            width: 36px;
            height: 36px;
            object-fit: contain;
        }

        .brand-title {
            font-size: 1.5rem;
            font-weight: 700;
            background: linear-gradient(135deg, #fff 0%, #a5b4fc 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            letter-spacing: -0.5px;
        }

        .status-badge {
            display: flex;
            align-items: center;
            justify-content: center;
            background: rgba(16, 185, 129, 0.1);
            border: 1px solid rgba(16, 185, 129, 0.2);
            width: 28px;
            height: 28px;
            border-radius: 50%;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            background-color: var(--accent-success);
            border-radius: 50%;
            box-shadow: 0 0 8px var(--accent-success);
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }
            70% { transform: scale(1); box-shadow: 0 0 0 6px rgba(16, 185, 129, 0); }
            100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2.5rem;
        }

        .stat-card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 16px;
            padding: 1.5rem;
            backdrop-filter: blur(12px);
            position: relative;
            overflow: hidden;
            transition: transform 0.3s ease, border-color 0.3s ease;
        }

        .stat-card:hover {
            transform: translateY(-4px);
            border-color: rgba(99, 102, 241, 0.3);
        }

        .stat-label {
            font-size: 0.9rem;
            color: var(--text-secondary);
            font-weight: 500;
            margin-bottom: 0.5rem;
        }

        .stat-value {
            font-size: 2.25rem;
            font-weight: 700;
            letter-spacing: -1px;
        }

        .charts-grid {
            display: grid;
            grid-template-columns: 1fr 2fr;
            gap: 1.5rem;
            margin-bottom: 2.5rem;
        }

        @media (max-width: 900px) {
            .charts-grid {
                grid-template-columns: 1fr;
            }
        }

        .chart-card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 16px;
            padding: 1.5rem;
            backdrop-filter: blur(12px);
            display: flex;
            flex-direction: column;
            gap: 1rem;
        }

        .chart-title {
            font-size: 1.1rem;
            font-weight: 600;
            color: var(--text-primary);
        }

        .chart-container {
            position: relative;
            width: 100%;
            height: 280px;
        }

        .history-card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 16px;
            padding: 1.5rem;
            backdrop-filter: blur(12px);
        }

        .history-title {
            font-size: 1.1rem;
            font-weight: 600;
            margin-bottom: 1.25rem;
        }

        table {
            width: 100%;
            border-collapse: collapse;
            text-align: left;
        }

        th {
            color: var(--text-secondary);
            font-weight: 600;
            font-size: 0.85rem;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            padding: 1rem;
            border-bottom: 1px solid var(--card-border);
        }

        td {
            padding: 1.2rem 1rem;
            border-bottom: 1px solid var(--card-border);
            font-size: 0.95rem;
            vertical-align: middle;
        }

        tr:last-child td {
            border-bottom: none;
        }

        tr:hover td {
            background: rgba(255, 255, 255, 0.01);
        }

        .badge {
            display: inline-block;
            padding: 0.25rem 0.6rem;
            border-radius: 6px;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .badge-text { background: rgba(99, 102, 241, 0.15); color: #a5b4fc; border: 1px solid rgba(99, 102, 241, 0.3); }
        .badge-image { background: rgba(20, 184, 166, 0.15); color: #99f6e4; border: 1px solid rgba(20, 184, 166, 0.3); }
        .badge-json { background: rgba(59, 130, 246, 0.15); color: #93c5fd; border: 1px solid rgba(59, 130, 246, 0.3); }
        .badge-sql { background: rgba(236, 72, 153, 0.15); color: #fbcfe8; border: 1px solid rgba(236, 72, 153, 0.3); }
        .badge-jwt { background: rgba(139, 92, 246, 0.15); color: #c4b5fd; border: 1px solid rgba(139, 92, 246, 0.3); }
        .badge-timestamp { background: rgba(245, 158, 11, 0.15); color: #fde68a; border: 1px solid rgba(245, 158, 11, 0.3); }

        .time-col {
            color: var(--text-secondary);
            font-size: 0.85rem;
            width: 160px;
        }

        .format-col {
            width: 120px;
        }

        .preview-text {
            max-width: 500px;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            font-family: monospace;
            color: #e2e8f0;
        }

        .thumbnail-preview {
            max-height: 50px;
            max-width: 120px;
            border-radius: 6px;
            border: 1px solid var(--card-border);
            object-fit: cover;
            display: block;
        }

        .empty-state {
            text-align: center;
            padding: 3rem;
            color: var(--text-secondary);
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="brand">
                <img class="brand-logo" src="/api/logo" alt="logo"/>
                <h1 class="brand-title">DevClip Dashboard</h1>
            </div>
            <div class="status-badge">
                <span class="status-dot"></span>
            </div>
        </header>

        <section class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Total Session Copies</div>
                <div class="stat-value" id="total-copies">0</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Text Items Count</div>
                <div class="stat-value" id="text-copies">0</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Image Items Count</div>
                <div class="stat-value" id="image-copies">0</div>
            </div>
        </section>

        <section class="charts-grid">
            <div class="chart-card">
                <h2 class="chart-title">Format Distribution</h2>
                <div class="chart-container">
                    <canvas id="format-chart"></canvas>
                </div>
            </div>
            <div class="chart-card">
                <h2 class="chart-title">Activity (Copies per Hour of Day)</h2>
                <div class="chart-container">
                    <canvas id="activity-chart"></canvas>
                </div>
            </div>
        </section>

        <section class="history-card">
            <h2 class="history-title">Recent Clipboard Logs</h2>
            <div style="overflow-x: auto;">
                <table>
                    <thead>
                        <tr>
                            <th class="time-col">Time</th>
                            <th class="format-col">Format</th>
                            <th>Content Preview</th>
                        </tr>
                    </thead>
                    <tbody id="history-tbody">
                        <tr>
                            <td colspan="3" class="empty-state">No clipboard logs recorded in this session yet.</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </section>
    </div>

    <script>
        let formatChartObj = null;
        let activityChartObj = null;

        function formatTime(dateStr) {
            const date = new Date(dateStr);
            return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        }

        function getFormatBadge(item) {
            if (item.kind === 1) {
                return '<span class="badge badge-image">IMAGE</span>';
            }
            const fmt = item.format || 'plain';
            switch (fmt.toLowerCase()) {
                case 'json': return '<span class="badge badge-json">JSON</span>';
                case 'sql': return '<span class="badge badge-sql">SQL</span>';
                case 'jwt': return '<span class="badge badge-jwt">JWT</span>';
                case 'timestamp': return '<span class="badge badge-timestamp">EPOCH</span>';
                default: return '<span class="badge badge-text">TEXT</span>';
            }
        }

        function updateCharts(stats) {
            const counts = stats.type_counts || {};
            const categories = [
                { key: 'plain', label: 'PLAIN', color: '#6366f1' },
                { key: 'json', label: 'JSON', color: '#3b82f6' },
                { key: 'sql', label: 'SQL', color: '#ec4899' },
                { key: 'jwt', label: 'JWT', color: '#8b5cf6' },
                { key: 'timestamp', label: 'EPOCH', color: '#f59e0b' },
                { key: 'image', label: 'IMAGE', color: '#14b8a6' }
            ];

            const labels = categories.map(c => c.label);
            const data = categories.map(c => counts[c.key] || 0);
            const colors = categories.map(c => c.color);

            if (formatChartObj) {
                formatChartObj.data.datasets[0].data = data;
                formatChartObj.update();
            } else {
                const ctx = document.getElementById('format-chart').getContext('2d');
                formatChartObj = new Chart(ctx, {
                    type: 'doughnut',
                    data: {
                        labels: labels,
                        datasets: [{
                            data: data,
                            backgroundColor: colors,
                            borderWidth: 0
                        }]
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false,
                        plugins: {
                            legend: {
                                position: 'bottom',
                                labels: { color: '#8e92b2', font: { family: 'Plus Jakarta Sans' } }
                            }
                        }
                    }
                });
            }

            const hourly = stats.copies_per_hour || Array(24).fill(0);
            const hoursLabels = Array.from({length: 24}, (_, i) => i + ':00');

            if (activityChartObj) {
                activityChartObj.data.datasets[0].data = hourly;
                activityChartObj.update();
            } else {
                const ctx = document.getElementById('activity-chart').getContext('2d');
                activityChartObj = new Chart(ctx, {
                    type: 'line',
                    data: {
                        labels: hoursLabels,
                        datasets: [{
                            label: 'Copies',
                            data: hourly,
                            borderColor: '#6366f1',
                            backgroundColor: 'rgba(99, 102, 241, 0.1)',
                            fill: true,
                            tension: 0.4,
                            borderWidth: 2
                        }]
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false,
                        plugins: {
                            legend: { display: false }
                        },
                        scales: {
                            x: {
                                grid: { color: 'rgba(255, 255, 255, 0.05)' },
                                ticks: { color: '#8e92b2' }
                            },
                            y: {
                                grid: { color: 'rgba(255, 255, 255, 0.05)' },
                                ticks: { color: '#8e92b2', stepSize: 1 }
                            }
                        }
                    }
                });
            }
        }

        async function fetchStats() {
            try {
                const res = await fetch('/api/stats');
                const data = await res.json();
                
                document.getElementById('total-copies').textContent = data.total_copies;
                
                const counts = data.type_counts || {};
                const imageCount = counts['image'] || 0;
                let textCount = 0;
                for (const [key, val] of Object.entries(counts)) {
                    if (key !== 'image') textCount += val;
                }
                document.getElementById('text-copies').textContent = textCount;
                document.getElementById('image-copies').textContent = imageCount;

                updateCharts(data);

                const tbody = document.getElementById('history-tbody');
                const history = data.history || [];
                if (history.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="3" class="empty-state">No clipboard logs recorded in this session yet.</td></tr>';
                } else {
                    tbody.innerHTML = history.map(item => {
                        let content = '';
                        if (item.kind === 1) {
                            content = '<img class="thumbnail-preview" src="/api/image?id=' + item.id + '" alt="thumbnail"/>';
                        } else {
                            const escaped = (item.preview || item.text || '')
                                .replace(/&/g, '&amp;')
                                .replace(/</g, '&lt;')
                                .replace(/>/g, '&gt;')
                                .replace(/"/g, '&quot;');
                            content = '<div class="preview-text">' + escaped + '</div>';
                        }
                        return '<tr>' +
                            '<td class="time-col">' + formatTime(item.createdAt) + '</td>' +
                            '<td class="format-col">' + getFormatBadge(item) + '</td>' +
                            '<td>' + content + '</td>' +
                            '</tr>';
                    }).join('');
                }
            } catch (err) {
                console.error("Error fetching stats:", err);
            }
        }

        fetchStats();
        setInterval(fetchStats, 1000);
    </script>
</body>
</html>
`
