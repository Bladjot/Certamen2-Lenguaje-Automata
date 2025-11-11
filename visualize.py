#!/usr/bin/env python3
"""
visualize.py - Generador de diagramas de espacio-tiempo para el simulador distribuido

Este script lee execution.csv y genera un diagrama de espacio-tiempo mostrando:
- Eje X: tiempo (timestamp)
- Eje Y: hilos (Scheduler, Workers)
- Colores distintos para cada tipo de acción
- Puntos y líneas para representar eventos

Requiere: matplotlib, pandas (instalar con: pip install matplotlib pandas)
"""

import csv
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
from collections import defaultdict
import sys
import os

# Configuración de colores para cada tipo de acción
ACTION_COLORS = {
    'Send': '#FF6B6B',        # Rojo suave
    'Receive': '#4ECDC4',     # Verde azulado
    'Checkpoint': '#45B7D1',  # Azul
    'Rollback': '#FFA07A',    # Naranja suave
    'Start': '#98D8C8',       # Verde menta
    'Process': '#FADCDC',     # Rosa claro
    'Straggler': '#FF1744',   # Rojo intenso
    'Execute': '#FFD93D',     # Amarillo
}

# Configuración de marcadores para cada tipo de acción
ACTION_MARKERS = {
    'Send': 'o',           # Círculo
    'Receive': 's',        # Cuadrado
    'Checkpoint': '^',     # Triángulo hacia arriba
    'Rollback': 'v',       # Triángulo hacia abajo
    'Start': 'D',          # Diamante
    'Process': 'p',        # Pentágono
    'Straggler': 'X',      # X grande
    'Execute': '+',        # Cruz
}

def read_execution_csv(csv_path):
    """
    Lee el archivo execution.csv y retorna los datos estructurados
    """
    print(f"Leyendo datos desde: {csv_path}")
    
    if not os.path.exists(csv_path):
        print(f"Error: Archivo {csv_path} no encontrado")
        return None
    
    data = []
    threads = set()
    
    try:
        with open(csv_path, 'r', newline='', encoding='utf-8') as file:
            reader = csv.DictReader(file)
            
            for row in reader:
                # Procesar cada fila
                thread_key = f"{row['ThreadType']}_{row['ThreadID']}"
                threads.add(thread_key)
                
                data.append({
                    'thread_key': thread_key,
                    'thread_type': row['ThreadType'],
                    'thread_id': int(row['ThreadID']),
                    'timestamp': int(row['Timestamp']),
                    'action': row['Action'],
                    'event_type': row['EventType'],
                    'lvt': int(row['LVT']),
                    'message': row['Message'],
                    'created_at': row['CreatedAt']
                })
    
    except Exception as e:
        print(f"Error leyendo CSV: {e}")
        return None
    
    print(f"Datos cargados: {len(data)} eventos, {len(threads)} threads")
    return data, sorted(list(threads))

def create_timeline_diagram(data, threads, output_path):
    """
    Crea el diagrama de espacio-tiempo
    """
    print("Generando diagrama de espacio-tiempo...")
    
    # Configurar el gráfico
    plt.figure(figsize=(16, max(8, len(threads) * 0.8)))
    plt.style.use('default')
    
    # Crear mapeo de threads a posiciones Y
    thread_positions = {thread: i for i, thread in enumerate(threads)}
    
    # Agrupar datos por acción para crear la leyenda
    actions_used = set()
    
    # Plotear cada evento
    for event in data:
        thread_key = event['thread_key']
        timestamp = event['timestamp']
        action = event['action']
        
        # Obtener posición Y del thread
        y_pos = thread_positions[thread_key]
        
        # Obtener color y marcador
        color = ACTION_COLORS.get(action, '#666666')  # Gris por defecto
        marker = ACTION_MARKERS.get(action, 'o')      # Círculo por defecto
        
        # Plotear punto
        plt.scatter(timestamp, y_pos, 
                   c=color, 
                   marker=marker, 
                   s=80,  # Tamaño del marcador
                   alpha=0.8,
                   edgecolors='black',
                   linewidth=0.5,
                   zorder=3)
        
        actions_used.add(action)
    
    # Conectar eventos del mismo thread con líneas suaves
    for thread in threads:
        thread_events = [e for e in data if e['thread_key'] == thread]
        thread_events.sort(key=lambda x: x['timestamp'])
        
        if len(thread_events) > 1:
            timestamps = [e['timestamp'] for e in thread_events]
            y_positions = [thread_positions[thread]] * len(timestamps)
            
            # Línea conectora suave
            plt.plot(timestamps, y_positions, 
                    color='lightgray', 
                    alpha=0.5, 
                    linewidth=1, 
                    zorder=1,
                    linestyle='-')
    
    # Configurar ejes
    plt.xlabel('Tiempo (Timestamp)', fontsize=12, fontweight='bold')
    plt.ylabel('Threads', fontsize=12, fontweight='bold')
    plt.title('Diagrama de Espacio-Tiempo - Simulador Distribuido Time Warp', 
              fontsize=14, fontweight='bold', pad=20)
    
    # Configurar eje Y con nombres de threads
    plt.yticks(range(len(threads)), threads, fontsize=10)
    
    # Configurar grilla
    plt.grid(True, alpha=0.3, linestyle='--', zorder=0)
    
    # Agregar líneas horizontales para separar threads
    for i in range(len(threads)):
        plt.axhline(y=i, color='lightgray', alpha=0.2, linewidth=0.5, zorder=0)
    
    # Crear leyenda
    legend_patches = []
    for action in sorted(actions_used):
        color = ACTION_COLORS.get(action, '#666666')
        marker = ACTION_MARKERS.get(action, 'o')
        
        # Crear patch para la leyenda
        patch = mpatches.Patch(color=color, label=f'{action} ({marker})')
        legend_patches.append(patch)
    
    plt.legend(handles=legend_patches, 
              loc='upper left', 
              bbox_to_anchor=(1.02, 1), 
              fontsize=10,
              title='Acciones',
              title_fontsize=11,
              framealpha=0.9)
    
    # Ajustar márgenes
    plt.tight_layout()
    
    # Guardar imagen
    plt.savefig(output_path, 
                dpi=300, 
                bbox_inches='tight', 
                facecolor='white',
                edgecolor='none')
    
    print(f"Diagrama guardado en: {output_path}")
    
    # Mostrar estadísticas
    if data:
        min_timestamp = min(e['timestamp'] for e in data)
        max_timestamp = max(e['timestamp'] for e in data)
        print(f"Rango temporal: {min_timestamp} - {max_timestamp}")
        print(f"Duración total: {max_timestamp - min_timestamp} unidades")
        print(f"Acciones detectadas: {', '.join(sorted(actions_used))}")

def create_detailed_timeline(data, threads, output_path):
    """
    Crea una versión más detallada del diagrama con información adicional
    """
    print("Generando diagrama detallado...")
    
    fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(16, 12), 
                                   gridspec_kw={'height_ratios': [3, 1]})
    
    # Thread positions
    thread_positions = {thread: i for i, thread in enumerate(threads)}
    
    # Diagrama principal (igual que antes)
    actions_used = set()
    
    for event in data:
        thread_key = event['thread_key']
        timestamp = event['timestamp']
        action = event['action']
        
        y_pos = thread_positions[thread_key]
        color = ACTION_COLORS.get(action, '#666666')
        marker = ACTION_MARKERS.get(action, 'o')
        
        ax1.scatter(timestamp, y_pos, 
                   c=color, 
                   marker=marker, 
                   s=80,
                   alpha=0.8,
                   edgecolors='black',
                   linewidth=0.5,
                   zorder=3)
        
        actions_used.add(action)
    
    # Conectar eventos
    for thread in threads:
        thread_events = [e for e in data if e['thread_key'] == thread]
        thread_events.sort(key=lambda x: x['timestamp'])
        
        if len(thread_events) > 1:
            timestamps = [e['timestamp'] for e in thread_events]
            y_positions = [thread_positions[thread]] * len(timestamps)
            
            ax1.plot(timestamps, y_positions, 
                    color='lightgray', 
                    alpha=0.5, 
                    linewidth=1, 
                    zorder=1)
    
    # Configurar primer subplot
    ax1.set_ylabel('Threads', fontsize=12, fontweight='bold')
    ax1.set_title('Diagrama de Espacio-Tiempo - Vista Principal', 
                  fontsize=14, fontweight='bold')
    ax1.set_yticks(range(len(threads)))
    ax1.set_yticklabels(threads, fontsize=10)
    ax1.grid(True, alpha=0.3, linestyle='--')
    
    # Segundo subplot: Histograma de actividad por tiempo
    timestamps = [e['timestamp'] for e in data]
    ax2.hist(timestamps, bins=50, alpha=0.7, color='skyblue', edgecolor='black', linewidth=0.5)
    ax2.set_xlabel('Tiempo (Timestamp)', fontsize=12, fontweight='bold')
    ax2.set_ylabel('Eventos', fontsize=12, fontweight='bold')
    ax2.set_title('Distribución de Actividad en el Tiempo', fontsize=12, fontweight='bold')
    ax2.grid(True, alpha=0.3, linestyle='--')
    
    # Leyenda para el diagrama principal
    legend_patches = []
    for action in sorted(actions_used):
        color = ACTION_COLORS.get(action, '#666666')
        marker = ACTION_MARKERS.get(action, 'o')
        patch = mpatches.Patch(color=color, label=f'{action} ({marker})')
        legend_patches.append(patch)
    
    ax1.legend(handles=legend_patches, 
              loc='upper left', 
              bbox_to_anchor=(1.02, 1), 
              fontsize=10,
              title='Acciones')
    
    plt.tight_layout()
    
    # Guardar imagen detallada
    detailed_path = output_path.replace('.png', '_detailed.png')
    plt.savefig(detailed_path, 
                dpi=300, 
                bbox_inches='tight', 
                facecolor='white')
    
    print(f"Diagrama detallado guardado en: {detailed_path}")

def generate_statistics_report(data, threads, output_path):
    """
    Genera un reporte estadístico en texto
    """
    report_path = output_path.replace('.png', '_stats.txt')
    
    print(f"Generando reporte estadístico en: {report_path}")
    
    with open(report_path, 'w', encoding='utf-8') as f:
        f.write("=== REPORTE ESTADÍSTICO - SIMULADOR DISTRIBUIDO ===\n\n")
        
        # Estadísticas generales
        f.write(f"Total de eventos: {len(data)}\n")
        f.write(f"Total de threads: {len(threads)}\n")
        
        if data:
            min_ts = min(e['timestamp'] for e in data)
            max_ts = max(e['timestamp'] for e in data)
            f.write(f"Rango temporal: {min_ts} - {max_ts}\n")
            f.write(f"Duración: {max_ts - min_ts} unidades\n\n")
        
        # Estadísticas por thread
        f.write("=== ESTADÍSTICAS POR THREAD ===\n")
        for thread in threads:
            thread_events = [e for e in data if e['thread_key'] == thread]
            f.write(f"\n{thread}:\n")
            f.write(f"  Total eventos: {len(thread_events)}\n")
            
            if thread_events:
                actions = defaultdict(int)
                for event in thread_events:
                    actions[event['action']] += 1
                
                f.write("  Acciones:\n")
                for action, count in sorted(actions.items()):
                    f.write(f"    {action}: {count}\n")
        
        # Estadísticas por acción
        f.write("\n=== ESTADÍSTICAS POR ACCIÓN ===\n")
        all_actions = defaultdict(int)
        for event in data:
            all_actions[event['action']] += 1
        
        for action, count in sorted(all_actions.items()):
            f.write(f"{action}: {count} eventos\n")

def main():
    """
    Función principal
    """
    print("=== VISUALIZADOR DE SIMULADOR DISTRIBUIDO ===")
    
    # Rutas de archivos
    csv_path = "execution.csv"
    output_path = "execution_timeline.png"
    
    # Permitir argumentos de línea de comandos
    if len(sys.argv) > 1:
        csv_path = sys.argv[1]
    if len(sys.argv) > 2:
        output_path = sys.argv[2]
    
    # Leer datos
    result = read_execution_csv(csv_path)
    if result is None:
        print("Error: No se pudieron cargar los datos")
        return 1
    
    data, threads = result
    
    if not data:
        print("Error: No hay datos para visualizar")
        return 1
    
    try:
        # Crear diagrama principal
        create_timeline_diagram(data, threads, output_path)
        
        # Crear diagrama detallado
        create_detailed_timeline(data, threads, output_path)
        
        # Generar reporte estadístico
        generate_statistics_report(data, threads, output_path)
        
        print("\n=== VISUALIZACIÓN COMPLETADA EXITOSAMENTE ===")
        print(f"Archivos generados:")
        print(f"  - {output_path} (diagrama principal)")
        print(f"  - {output_path.replace('.png', '_detailed.png')} (diagrama detallado)")
        print(f"  - {output_path.replace('.png', '_stats.txt')} (estadísticas)")
        
        return 0
        
    except Exception as e:
        print(f"Error generando visualización: {e}")
        return 1

if __name__ == "__main__":
    exit(main())
