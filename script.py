import json
import numpy as np
import matplotlib.pyplot as plt
from mpl_toolkits.mplot3d import Axes3D
import os
import sys
from typing import Dict, List, Optional, Tuple

def load_simulation_data(filename: str) -> Optional[List[dict]]:
    try:
        with open(filename, 'r') as f:
            data = json.load(f)
            if not isinstance(data, list) or len(data) == 0:
                raise ValueError("Invalid simulation data format")
            print(f"Loaded {len(data)} snapshots from {filename}")
            return data
    except Exception as e:
        print(f"Error loading simulation data: {e}")
        return None

def validate_particle_data(particle: dict) -> bool:
    """Validate particle data structure and values."""
    try:
        # Check position
        pos = particle.get('Position', {})
        vel = particle.get('Velocity', {})
        if not all(isinstance(pos.get(k), (int, float)) for k in ['X', 'Y', 'Z']):
            return False
        if not all(isinstance(vel.get(k), (int, float)) for k in ['X', 'Y', 'Z']):
            return False
        if not isinstance(particle.get('Temperature'), (int, float)):
            return False
        return True
    except Exception:
        return False

def analyze_flow_field(data: List[dict]) -> Optional[Dict[str, np.ndarray]]:
    if not data or len(data) < 2:
        print("Insufficient simulation data (need at least 2 snapshots)")
        return None
    
    print(f"\nAnalyzing simulation data:")
    print(f"Number of snapshots: {len(data)}")
    
    # Extract time series data
    times = []
    velocities = []
    positions = []
    
    try:
        for i, snapshot in enumerate(data):
            if not isinstance(snapshot, dict):
                print(f"Invalid snapshot format at index {i}")
                continue
                
            timestamp = snapshot.get('Timestamp', 0)
            particles = snapshot.get('Particles', [])
            
            if not particles:
                print(f"No particles in snapshot {i}")
                continue
            
            # Calculate average velocity and position for this snapshot
            valid_particles = [p for p in particles if validate_particle_data(p)]
            
            if not valid_particles:
                print(f"No valid particles in snapshot {i}")
                continue
                
            # Calculate averages using numpy for efficiency
            positions_array = np.array([[p['Position']['X'], p['Position']['Y'], p['Position']['Z']] 
                                     for p in valid_particles])
            velocities_array = np.array([[p['Velocity']['X'], p['Velocity']['Y'], p['Velocity']['Z']] 
                                       for p in valid_particles])
            
            avg_pos = positions_array.mean(axis=0)
            avg_vel = velocities_array.mean(axis=0)
            
            times.append(timestamp)
            positions.append(avg_pos)
            velocities.append(avg_vel)
            
            if i == 0 or i == len(data)-1:
                print(f"\nSnapshot {i} (t={timestamp:.2f}s):")
                print(f"Number of valid particles: {len(valid_particles)}")
                print(f"Avg position = ({avg_pos[0]:.2f}, {avg_pos[1]:.2f}, {avg_pos[2]:.2f})")
                print(f"Avg velocity = ({avg_vel[0]:.2f}, {avg_vel[1]:.2f}, {avg_vel[2]:.2f})")
        
        if len(times) < 2:
            print("Not enough valid time points for visualization")
            return None
            
        # Convert to numpy arrays
        times = np.array(times)
        velocities = np.array(velocities)
        positions = np.array(positions)
        
        print("\nProcessed data summary:")
        print(f"Time points: {len(times)}")
        print(f"Time range: {times[0]:.2f}s to {times[-1]:.2f}s")
        print(f"Position range: {positions.min():.2f} to {positions.max():.2f}")
        print(f"Velocity range: {velocities.min():.2f} to {velocities.max():.2f}")
        
        return {
            'times': times,
            'velocities': velocities,
            'positions': positions
        }
        
    except Exception as e:
        print(f"Error processing simulation data: {e}")
        import traceback
        traceback.print_exc()
        return None

def create_velocity_components_plot(data: Dict[str, np.ndarray], output_path: str) -> None:
    try:
        plt.figure(figsize=(12, 8))
        plt.plot(data['times'], data['velocities'][:, 0], 'r-', label='X', linewidth=2)
        plt.plot(data['times'], data['velocities'][:, 1], 'g-', label='Y', linewidth=2)
        plt.plot(data['times'], data['velocities'][:, 2], 'b-', label='Z', linewidth=2)
        plt.title('Particle Velocity Components Over Time', fontsize=14)
        plt.xlabel('Time (s)', fontsize=12)
        plt.ylabel('Velocity (m/s)', fontsize=12)
        plt.legend(fontsize=12)
        plt.grid(True)
        plt.tight_layout()
        plt.savefig(output_path, dpi=300, bbox_inches='tight')
        plt.close()
        print(f"Created velocity components plot: {output_path}")
    except Exception as e:
        print(f"Error creating velocity plot: {e}")

def create_velocity_magnitude_plot(data: Dict[str, np.ndarray], output_path: str) -> None:
    try:
        plt.figure(figsize=(12, 8))
        velocity_magnitudes = np.sqrt(np.sum(data['velocities']**2, axis=1))
        plt.plot(data['times'], velocity_magnitudes, 'k-', linewidth=2)
        plt.title('Particle Speed Over Time', fontsize=14)
        plt.xlabel('Time (s)', fontsize=12)
        plt.ylabel('Speed (m/s)', fontsize=12)
        plt.grid(True)
        plt.tight_layout()
        plt.savefig(output_path, dpi=300, bbox_inches='tight')
        plt.close()
        print(f"Created velocity magnitude plot: {output_path}")
    except Exception as e:
        print(f"Error creating magnitude plot: {e}")

def create_trajectory_plot(data: Dict[str, np.ndarray], output_path: str) -> None:
    try:
        positions = data['positions']
        if len(positions) < 2:
            print("Not enough position data for trajectory plot")
            return

        fig = plt.figure(figsize=(14, 10))
        ax = fig.add_subplot(111, projection='3d')
        
        # Plot trajectory line with gradient color based on time
        points = ax.scatter(positions[:, 0], positions[:, 1], positions[:, 2],
                          c=data['times'], cmap='viridis',
                          s=30, alpha=0.6)
        fig.colorbar(points, label='Time (s)')
        
        # Plot trajectory line
        ax.plot3D(positions[:, 0], positions[:, 1], positions[:, 2],
                 'b-', linewidth=2, alpha=0.4)
        
        # Plot start and end points
        ax.scatter(positions[0, 0], positions[0, 1], positions[0, 2],
                  c='green', marker='o', s=200, label='Start', alpha=0.8)
        ax.scatter(positions[-1, 0], positions[-1, 1], positions[-1, 2],
                  c='red', marker='o', s=200, label='End', alpha=0.8)
        
        ax.set_title('Particle Trajectory', fontsize=14)
        ax.set_xlabel('X Position (m)', fontsize=12)
        ax.set_ylabel('Y Position (m)', fontsize=12)
        ax.set_zlabel('Z Position (m)', fontsize=12)
        ax.legend(fontsize=12)
        
        # Set equal aspect ratio
        max_range = np.array([
            positions[:, 0].max() - positions[:, 0].min(),
            positions[:, 1].max() - positions[:, 1].min(),
            positions[:, 2].max() - positions[:, 2].min()
        ]).max() / 2.0
        
        mean_x = positions[:, 0].mean()
        mean_y = positions[:, 1].mean()
        mean_z = positions[:, 2].mean()
        
        ax.set_xlim(mean_x - max_range, mean_x + max_range)
        ax.set_ylim(mean_y - max_range, mean_y + max_range)
        ax.set_zlim(mean_z - max_range, mean_z + max_range)
        
        ax.grid(True)
        plt.tight_layout()
        plt.savefig(output_path, dpi=300, bbox_inches='tight')
        plt.close()
        print(f"Created trajectory plot: {output_path}")
    except Exception as e:
        print(f"Error creating trajectory plot: {e}")

def create_position_components_plot(data: Dict[str, np.ndarray], output_path: str) -> None:
    try:
        plt.figure(figsize=(12, 8))
        plt.plot(data['times'], data['positions'][:, 0], 'r-', label='X', linewidth=2)
        plt.plot(data['times'], data['positions'][:, 1], 'g-', label='Y', linewidth=2)
        plt.plot(data['times'], data['positions'][:, 2], 'b-', label='Z', linewidth=2)
        plt.title('Particle Position Components Over Time', fontsize=14)
        plt.xlabel('Time (s)', fontsize=12)
        plt.ylabel('Position (m)', fontsize=12)
        plt.legend(fontsize=12)
        plt.grid(True)
        plt.tight_layout()
        plt.savefig(output_path, dpi=300, bbox_inches='tight')
        plt.close()
        print(f"Created position components plot: {output_path}")
    except Exception as e:
        print(f"Error creating position plot: {e}")

def main():
    if len(sys.argv) != 2:
        print("Usage: python script.py <simulation_data.json>")
        sys.exit(1)

    input_file = sys.argv[1]
    base_path = os.path.splitext(input_file)[0]

    # Load and process data
    raw_data = load_simulation_data(input_file)
    if not raw_data:
        print("Failed to load simulation data")
        sys.exit(1)

    # Analyze the data
    processed_data = analyze_flow_field(raw_data)
    if not processed_data:
        print("Failed to analyze simulation data")
        sys.exit(1)

    # Create individual plot files
    plot_files = {
        'velocity': f"{base_path}_velocity.png",
        'magnitude': f"{base_path}_magnitude.png",
        'trajectory': f"{base_path}_trajectory.png",
        'position': f"{base_path}_position.png"
    }

    # Generate each plot
    create_velocity_components_plot(processed_data, plot_files['velocity'])
    create_velocity_magnitude_plot(processed_data, plot_files['magnitude'])
    create_trajectory_plot(processed_data, plot_files['trajectory'])
    create_position_components_plot(processed_data, plot_files['position'])

    print("\nVisualization complete. Files created:")
    for plot_type, filepath in plot_files.items():
        print(f"- {plot_type}: {filepath}")

if __name__ == "__main__":
    main()