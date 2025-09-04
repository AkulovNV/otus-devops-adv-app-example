#!/usr/bin/env python3
"""
Test Web Application with Prometheus Metrics
Демонстрирует все 4 типа метрик Prometheus: Counter, Gauge, Histogram, Summary
"""

import time
import random
import threading
from flask import Flask, request, jsonify
from prometheus_client import Counter, Gauge, Histogram, Summary, generate_latest, CONTENT_TYPE_LATEST
import psutil
import os

app = Flask(__name__)

# =============================================================================
# PROMETHEUS МЕТРИКИ - Все 4 типа
# =============================================================================

# COUNTER - накопительные метрики
http_requests_total = Counter(
    'http_requests_total',
    'Total number of HTTP requests',
    ['method', 'endpoint', 'status']
)

http_errors_total = Counter(
    'http_errors_total', 
    'Total number of HTTP errors',
    ['error_type']
)

# GAUGE - текущие значения
active_connections = Gauge(
    'http_active_connections',
    'Number of active HTTP connections'
)

system_memory_usage = Gauge(
    'system_memory_usage_bytes',
    'Current memory usage in bytes'
)

system_cpu_usage = Gauge(
    'system_cpu_usage_percent',
    'Current CPU usage percentage'
)

application_queue_size = Gauge(
    'application_queue_size',
    'Current number of items in processing queue'
)

# HISTOGRAM - распределение значений  
http_request_duration_seconds = Histogram(
    'http_request_duration_seconds',
    'HTTP request duration in seconds',
    ['endpoint'],
    buckets=[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
)

request_size_bytes = Histogram(
    'http_request_size_bytes',
    'HTTP request size in bytes',
    buckets=[100, 1000, 5000, 10000, 50000, 100000, 500000]
)

# SUMMARY - квантили на клиенте
response_time_summary = Summary(
    'http_response_time_summary_seconds',
    'Response time summary',
    ['endpoint']
)

# =============================================================================
# BACKGROUND TASKS для realistic метрик
# =============================================================================

def update_system_metrics():
    """Background task для обновления system метрик"""
    while True:
        try:
            # Обновляем system метрики
            system_memory_usage.set(psutil.virtual_memory().used)
            system_cpu_usage.set(psutil.cpu_percent())
            
            # Симулируем изменения queue size
            current_queue = application_queue_size._value._value
            change = random.randint(-5, 10)
            new_size = max(0, current_queue + change)
            application_queue_size.set(new_size)
            
        except Exception as e:
            http_errors_total.labels(error_type='system_metrics').inc()
            
        time.sleep(5)

def simulate_background_load():
    """Симулирует background нагрузку для realistic метрик"""
    while True:
        # Симулируем batch processing
        processing_time = random.uniform(0.1, 2.0)
        with request_size_bytes.time():
            time.sleep(processing_time)
            
        # Периодически создаем искусственные ошибки
        if random.random() < 0.02:  # 2% chance
            http_errors_total.labels(error_type='background_task').inc()
            
        time.sleep(random.uniform(1, 5))

# Запускаем background tasks
threading.Thread(target=update_system_metrics, daemon=True).start()
threading.Thread(target=simulate_background_load, daemon=True).start()

# =============================================================================
# HTTP ENDPOINTS
# =============================================================================

@app.before_request
def before_request():
    """Tracking активных соединений"""
    active_connections.inc()

@app.after_request  
def after_request(response):
    """Обновляем метрики после каждого запроса"""
    active_connections.dec()
    
    # Записываем Counter метрики
    http_requests_total.labels(
        method=request.method,
        endpoint=request.endpoint or 'unknown',
        status=response.status_code
    ).inc()
    
    return response

@app.route('/')
def home():
    """Главная страница с базовой латенсей"""
    start_time = time.time()
    
    # Симулируем нормальную обработку
    time.sleep(random.uniform(0.01, 0.05))
    
    # Записываем Histogram и Summary
    duration = time.time() - start_time
    http_request_duration_seconds.labels(endpoint='home').observe(duration)
    response_time_summary.labels(endpoint='home').observe(duration)
    
    # Симулируем размер request
    request_size_bytes.observe(random.randint(500, 2000))
    
    return jsonify({
        'status': 'healthy',
        'service': 'test-web-app',
        'version': '1.0.0',
        'timestamp': time.time()
    })

@app.route('/api/data')
def api_data():
    """API endpoint с variable латенсей"""
    start_time = time.time()
    
    # Симулируем обработку данных (переменное время)
    processing_time = random.uniform(0.05, 0.3)
    time.sleep(processing_time)
    
    # 5% chance медленного запроса  
    if random.random() < 0.05:
        time.sleep(random.uniform(1.0, 3.0))
        
    duration = time.time() - start_time
    http_request_duration_seconds.labels(endpoint='api_data').observe(duration)
    response_time_summary.labels(endpoint='api_data').observe(duration)
    
    # Больший размер request для API
    request_size_bytes.observe(random.randint(2000, 10000))
    
    return jsonify({
        'data': [{'id': i, 'value': random.random()} for i in range(10)],
        'processing_time': duration,
        'queue_size': application_queue_size._value._value
    })

@app.route('/api/slow')  
def slow_endpoint():
    """Намеренно медленный endpoint"""
    start_time = time.time()
    
    # Всегда медленный ответ
    time.sleep(random.uniform(2.0, 5.0))
    
    duration = time.time() - start_time
    http_request_duration_seconds.labels(endpoint='slow_api').observe(duration)
    response_time_summary.labels(endpoint='slow_api').observe(duration)
    
    return jsonify({'message': 'This endpoint is intentionally slow'})

@app.route('/api/error')
def error_endpoint():
    """Endpoint для генерации ошибок"""
    # 70% chance ошибки
    if random.random() < 0.7:
        http_errors_total.labels(error_type='intentional_500').inc()
        return jsonify({'error': 'Internal server error'}), 500
    
    # 20% chance 404
    if random.random() < 0.2:
        http_errors_total.labels(error_type='not_found').inc()
        return jsonify({'error': 'Not found'}), 404
        
    # 10% chance успех
    return jsonify({'message': 'Success (rare!)'})

@app.route('/health')
def health_check():
    """Health check endpoint (быстрый ответ)"""
    start_time = time.time()
    
    # Минимальная латенсь
    time.sleep(0.001)
    
    duration = time.time() - start_time
    http_request_duration_seconds.labels(endpoint='health').observe(duration)
    
    return jsonify({
        'status': 'healthy',
        'uptime': time.time() - app_start_time,
        'active_connections': active_connections._value._value,
        'queue_size': application_queue_size._value._value
    })

@app.route('/metrics')
def metrics():
    """Prometheus metrics endpoint"""
    return generate_latest(), 200, {'Content-Type': CONTENT_TYPE_LATEST}

@app.route('/load-test')
def load_test():
    """Endpoint для нагрузочного тестирования"""
    start_time = time.time()
    
    # Различные сценарии нагрузки
    scenario = random.choice(['light', 'medium', 'heavy'])
    
    if scenario == 'light':
        time.sleep(random.uniform(0.01, 0.05))
    elif scenario == 'medium':  
        time.sleep(random.uniform(0.1, 0.5))
    else:  # heavy
        time.sleep(random.uniform(1.0, 2.0))
        
    duration = time.time() - start_time
    http_request_duration_seconds.labels(endpoint='load_test').observe(duration)
    response_time_summary.labels(endpoint='load_test').observe(duration)
    
    # Большой размер response для heavy scenario
    response_size = 1000 if scenario == 'heavy' else 100
    request_size_bytes.observe(response_size)
    
    return jsonify({
        'scenario': scenario,
        'duration': duration,
        'timestamp': time.time()
    })

# =============================================================================
# ДОПОЛНИТЕЛЬНЫЕ УТИЛИТЫ
# =============================================================================

@app.route('/simulate-anomaly')
def simulate_anomaly():
    """Создает anomaly для демонстрации detection"""
    # Резко увеличиваем queue size
    application_queue_size.set(random.randint(500, 1000))
    
    # Генерируем burst ошибок
    for _ in range(10):
        http_errors_total.labels(error_type='anomaly_burst').inc()
        
    return jsonify({'message': 'Anomaly simulated - check your dashboards!'})

@app.route('/reset-anomaly')  
def reset_anomaly():
    """Сбрасываем anomaly"""
    application_queue_size.set(random.randint(5, 20))
    return jsonify({'message': 'Anomaly cleared'})

@app.route('/debug/metrics-info')
def debug_metrics():
    """Debug информация о метриках"""
    return jsonify({
        'counter_samples': {
            'http_requests_total': http_requests_total._value._value,
            'http_errors_total': {k: v._value._value for k, v in http_errors_total._metrics.items()}
        },
        'gauge_values': {
            'active_connections': active_connections._value._value,
            'memory_usage': system_memory_usage._value._value,
            'cpu_usage': system_cpu_usage._value._value,
            'queue_size': application_queue_size._value._value
        },
        'histogram_buckets': len(http_request_duration_seconds._upper_bounds),
        'summary_quantiles': [0.5, 0.9, 0.99]
    })

# =============================================================================
# APPLICATION STARTUP
# =============================================================================

if __name__ == '__main__':
    app_start_time = time.time()
    
    # Инициализация метрик
    application_queue_size.set(random.randint(5, 20))
    active_connections.set(0)
    
    print("🚀 Test Web App starting...")
    print("📊 Metrics available at /metrics")
    print("🔍 Debug info at /debug/metrics-info")
    print("⚡ Endpoints:")
    print("   GET  /              - Home page (fast)")
    print("   GET  /api/data      - API with variable latency") 
    print("   GET  /api/slow      - Intentionally slow API")
    print("   GET  /api/error     - Error generator (70% fail rate)")
    print("   GET  /health        - Health check (minimal latency)")
    print("   GET  /load-test     - Load testing scenarios")
    print("   POST /simulate-anomaly  - Create anomaly for detection")
    print("   POST /reset-anomaly     - Clear anomaly")
    
    # Запускаем приложение
    app.run(
        host='0.0.0.0', 
        port=int(os.environ.get('PORT', 8080)),
        debug=False,
        threaded=True
    )