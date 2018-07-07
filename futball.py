#!/usr/bin/env python

import numpy as np
import pygame
import sys
import math
import random
import argparse

try:
    import _engine
    from futhark_ffi.compat import FutharkCompat
    print('Using futhark-pycffi backend.')
    def futhark_object():
        return FutharkCompat(_engine)
except ImportError:
    import engine
    print('Using futhark-pyopencl backend.')
    def futhark_object():
        return engine.engine(interactive=True)

class Ball:
    def __init__(self, engine, radius, colour, shine, pos, trajectory):
        self.engine = engine
        self.radius = radius
        self.colour = colour
        self.shine = shine
        self.pos = pos
        self.trajectory = trajectory

    def distance_to(self, point):
        return np.linalg.norm(self.pos-point)

    def insert_in_world(self, world):
        world, self.index = self.engine.add_sphere(world,
                                                   self.pos[0], self.pos[1], self.pos[2],
                                                   self.radius, self.colour, self.shine)
        return world

class FutballQuit(Exception):
    pass

class FutballDead(Exception):
    pass

class FutballGUI:
    def __init__ (self, args):
        bouncelimit=5
        movespeed=1000
        rotspeed=math.pi
        max_ball_dist=10000
        player_height=100

        self.width = args.width
        self.height = args.height
        self.fov = args.fov
        self.bouncelimit = 5
        self.movespeed = movespeed
        self.rotspeed = rotspeed
        self.floor_y = 0
        self.player_height = 100
        self.max_ball_dist = max_ball_dist
        self.seconds_per_ball = 3
        self.target_fps = args.fps

        self.white=0xffffff
        self.red=0xff0000
        self.green=0x00ff00
        self.blue=0x0000ff
        self.yellow=0xffff00
        self.ambient=self.white
        self.ambient_intensity=0

        self.engine = futhark_object()

        size=(self.width, self.height)
        pygame.init()
        pygame.display.set_caption('Futball!')
        self.screen = pygame.display.set_mode(size)
        self.surface = pygame.Surface(size, depth=32)
        self.font = pygame.font.Font(None, 36)
        pygame.key.set_repeat(500, 50)
        pygame.event.set_grab(True)
        pygame.mouse.set_visible(False)
        self.clock = pygame.time.Clock()
        self.roll = pygame.mixer.Sound('assets/roll.ogg')
        pygame.mixer.music.load('assets/music.ogg')

    def run(self):
        pygame.mixer.music.play(-1)
        self.most_survived = 0
        while True:
            pygame.mixer.set_num_channels(0)
            self.balls = []
            self.until_ball = 0
            self.eye = {'point': np.array([0, self.player_height, 0], dtype=np.float32),
                        'dir': np.array([0, 0], dtype=np.float32)}
            world = self.engine.empty_world()
            world, _ = self.engine.add_plane(world, 0, self.floor_y, 0, 0, 1, 0, self.white, 0.2)
            world = self.engine.add_light(world,  2000, 1000,  0,    self.red,    1)
            world = self.engine.add_light(world, -2000, 3000,  0,    self.green,  1)
            world = self.engine.add_light(world, 0,     1000,  2000, self.blue,   1)
            world = self.engine.add_light(world, 0,     1000, -2000, self.yellow, 1)
            self.world = world

            self.in_jump=False
            self.trajectory=np.array([0.0, 0.0, 0.0], dtype=np.float32)

            try:
                while True:
                    self.tick()
            except FutballQuit:
                return
            except FutballDead:
                self.most_survived = max(self.most_survived, len(self.balls))

    def random_ball(self):
            angle = random.random() * 2 * math.pi
            dist = 2000 + random.random() * 5000
            speed = 0.3 + random.random()*0.7/10
            radius = random.random() * self.player_height * 3
            x = np.cos(angle) * dist
            z = np.sin(angle) * dist
            rel_pos = np.array([x, 0, z], dtype=np.float32)
            pos = np.array([self.eye['point'][0] + rel_pos[0],
                            self.floor_y + radius,
                            self.eye['point'][2] + rel_pos[2]], dtype=np.float32)
            return Ball(self.engine, radius, self.white, 0.8, pos, -rel_pos*speed)

    def update_ball_positions(self, tdelta):
        num_balls = len(self.balls)
        xs = np.ndarray(num_balls, dtype=np.float32)
        ys = np.ndarray(num_balls, dtype=np.float32)
        zs = np.ndarray(num_balls, dtype=np.float32)

        for i in range(num_balls):
            b = self.balls[i]
            if b.distance_to(self.eye['point']) > self.max_ball_dist:
                b = self.random_ball()
                self.world = self.engine.set_sphere_radius(self.world, i, b.radius)
                self.balls[i] = b
            b.pos += b.trajectory * tdelta
            xs[i], ys[i], zs[i] = b.pos

        self.world = self.engine.set_sphere_positions(self.world, xs, ys, zs)

    def maybe_insert_ball(self, delta):
        self.until_ball -= delta
        if self.until_ball <= 0:
            b = self.random_ball()
            self.balls.append(b)
            pygame.mixer.set_num_channels(len(self.balls))
            pygame.mixer.Channel(len(self.balls)-1).play(self.roll, loops=-1)
            self.world = b.insert_in_world(self.world)
            self.until_ball = self.seconds_per_ball

    def check_for_collisions(self):
        # Check for ball-ball collisions.  This is done with a naive
        # O(n**2) algorithm.  If we want to scale to thousands of
        # balls, we could move this to Futhark.  For now, I think
        # Python is fine.
        num_balls = len(self.balls)
        for i in range(num_balls):
            for j in range(num_balls-1-i):
                b1 = self.balls[i]
                b2 = self.balls[i+j+1]
                if np.linalg.norm(b1.pos - b2.pos) < b1.radius+b2.radius:
                    # Collision!  Here comes a hack because I don't
                    # remember my vector calculus well enough.
                    x_diff = np.abs(b1.trajectory[0] - b2.trajectory[0])
                    z_diff = np.abs(b1.trajectory[2] - b2.trajectory[2])
                    if x_diff > z_diff:
                        b1.trajectory[0] *= -1
                        b2.trajectory[0] *= -1
                    else:
                        b1.trajectory[2] *= -1
                        b2.trajectory[2] *= -1

        # Check whether the fool player couldn't move fast enough.
        for b in self.balls:
            feet = np.array([self.eye['point'][0], b.radius, self.eye['point'][2]])
            if b.distance_to(feet) < b.radius or b.distance_to(self.eye['point']) < b.radius :
                print("You got to {} balls, but now you're dead.".format(len(self.balls)))
                raise FutballDead()

    def adjust_sounds(self):
        dropoff=2000
        closeness = 0.1
        for i in range(len(self.balls)):
            b = self.balls[i]
            closeness = np.log2(dropoff/np.linalg.norm(self.eye['point']-b.pos))
            pygame.mixer.Channel(i).set_volume(max(0.1, closeness))

    def render(self):
        def show_text(what, where):
            text = self.font.render(what, 1, (255, 255, 255))
            self.screen.blit(text, where)

        frame = self.engine.render(self.world, self.width, self.height, self.fov,
                                   self.eye['point'][0], self.eye['point'][1], self.eye['point'][2],
                                   self.eye['dir'][0], self.eye['dir'][1],
                                   self.ambient, self.ambient_intensity,
                                   self.bouncelimit).get()
        pygame.surfarray.blit_array(self.surface, frame)
        self.screen.blit(self.surface, (0, 0))

        speedmessage = "FPS: %.2f (%d bounces)" % (self.clock.get_fps(), self.bouncelimit)
        show_text(speedmessage, (10, 10))
        locmessage = ("Balls: %d   Next in: %.2fs   Most survived: %d" %
                      (len(self.balls), self.until_ball, self.most_survived))
        show_text(locmessage, (10, 40))

        pygame.display.flip()

    def handle_movement(self, delta):
        delta_x, delta_y = pygame.mouse.get_rel()
        self.eye['dir'][0] += float(delta_x)/self.width
        self.eye['dir'][1] += float(delta_y)/self.height
        self.eye['dir'][0] %= (math.pi*2)
        self.eye['dir'][1] = min(max(self.eye['dir'][1], -math.pi/2+0.001), math.pi/2-0.001)

        if self.in_jump:
            self.trajectory[1] -= 9.8 * 500 * delta
        else:
            self.trajectory[0] = 0
            self.trajectory[2] = 0

        if self.eye['point'][1] < self.floor_y + self.player_height:
            self.eye['point'][1] = self.floor_y + self.player_height
            self.trajectory[0] = 0
            self.trajectory[1] = 0
            self.trajectory[2] = 0
            self.in_jump = False


        def forwards(amount):
            a = self.eye['dir'][0]
            return np.array([amount * math.cos(a), 0, amount * math.sin(a)])

        def sideways(amount):
            a = self.eye['dir'][0] + math.pi/2
            return np.array([amount * math.cos(a), 0, amount * math.sin(a)])

        pressed = pygame.key.get_pressed()
        if pressed[pygame.K_a] and not self.in_jump:
            self.trajectory += sideways(-self.movespeed)
        if pressed[pygame.K_d] and not self.in_jump:
            self.trajectory += sideways(self.movespeed)
        if (pressed[pygame.K_w] or pressed[pygame.K_UP]) and not self.in_jump:
            self.trajectory += forwards(self.movespeed)
        if (pressed[pygame.K_s] or pressed[pygame.K_DOWN]) and not self.in_jump:
            self.trajectory += forwards(-self.movespeed)
        if pressed[pygame.K_SPACE] and not self.in_jump:
            self.trajectory[1] = 1500
            self.in_jump=True
        if pressed[pygame.K_RIGHT]:
            self.eye['dir'][0] = (self.eye['dir'][0] + self.rotspeed*delta) % (math.pi*2)
        if pressed[pygame.K_LEFT]:
            self.eye['dir'][0] = (self.eye['dir'][0] - self.rotspeed*delta) % (math.pi*2)

        self.eye['point'] += self.trajectory * delta

    def handle_other_input(self):
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                raise FluidQuit()
            elif event.type == pygame.KEYDOWN:
                if event.key == pygame.K_ESCAPE:
                    sys.exit()
                if event.unicode == 'z':
                    self.bouncelimit = max(bouncelimit-1, 1)
                if event.unicode == 'x':
                    self.bouncelimit += 1

    def tick(self):
        delta = self.clock.tick(self.target_fps) / 1000.0

        self.maybe_insert_ball(delta)

        self.check_for_collisions()

        self.update_ball_positions(delta)

        self.adjust_sounds()

        self.handle_movement(delta)

        self.handle_other_input()

        self.render()

def main():
    parser = argparse.ArgumentParser(description='FUTBALL')
    parser.add_argument('--fps', metavar='N', type=int,
                        help='cap FPS to this number', default=60)
    parser.add_argument('--fov', metavar='D', type=int,
                        help='field of view', default=105)
    parser.add_argument('--width', metavar='N', type=int,
                        help='width of the window in pixels', default=1280)
    parser.add_argument('--height', metavar='N', type=int,
                        help='height of the window in pixels', default=720)

    args = parser.parse_args()

    f = FutballGUI(args)
    f.run()
    return 0

if __name__ == '__main__':
    sys.exit(main())
