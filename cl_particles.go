package main

var kernel = `
typedef struct Particle {
	double vX, vY, vZ;
	double lX, lY, lZ;
	double Mass;
} Particle;

__kernel void updatepart(__global Particle *ps) {
	int i = get_global_id(0);
	double G = 3.0, dt = 0.1;
	for (int j = 0; j < NUM_PARTICLES; j++) {
		if (i == j) continue;
		double dx = ps[i].lX - ps[j].lX;
		double dy = ps[i].lY - ps[j].lY;
		double dz = ps[i].lZ - ps[j].lZ;
		double dist = sqrt(dx*dx + dy*dy + dz*dz);
		if (dist < 0.000001) dist = 0.000001;
		double acc = G * ps[j].Mass / (dist * dist);
		double factor = dt * acc / dist;
		ps[i].vX = dx * acc;
		ps[i].vY += dy * acc;
		ps[i].vZ += dz * acc;
	}
	`
