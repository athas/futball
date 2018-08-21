import "lib/github.com/athas/matte/colour"

import "types"
import "objects"
import "intersection"
import "lights"

let cast_view_rays (sizeX: i32) (sizeY: i32) (fov: i32) (eye_dir: position)
                 : [sizeY][sizeX]direction =
  let eye_vector = vec3.(normalise eye_dir)
  let vp_right = vec3.normalise (vec3.cross eye_vector {x=0,y=1,z=0})
  let vp_up = vec3.normalise (vec3.cross vp_right eye_vector)
  let fov_radians = f32.pi * (r32 fov / 2) / 180
  let height_width_ratio = r32 sizeY / r32 sizeX
  let half_width = f32.tan fov_radians
  let half_height = height_width_ratio * half_width
  let camera_width = half_width * 2
  let camera_height = half_height * 2
  let pixel_width = camera_width / (r32 sizeX - 1)
  let pixel_height = camera_height / (r32 sizeY - 1)

  let cast (x: i32) (y: i32) =
    let xcomp = vec3.scale ((r32 x * pixel_width) - half_width) vp_right
    let ycomp = vec3.scale ((r32 y * pixel_height) - half_height) vp_up
    in vec3.(normalise (eye_vector + xcomp + ycomp))
  in map (\y -> map (`cast` y) (iota sizeX)) (reverse (iota sizeY))

let hit_sphere (sph: sphere) (dist: f32) (orig: position) (dir: direction)
             : (position, direction, argb.colour, f32) =
  let point = orig vec3.+ vec3.scale dist dir
  let normal = sphere_normal sph point
  let colour = sph.colour
  let shine = sph.shine
  in (point, normal, colour, shine)

let hit_plane (pln: plane) (dist: f32) (orig: position) (dir: direction)
            : (position, direction, argb.colour, f32) =
  let point = orig vec3.+ vec3.scale dist dir
  let normal = pln.normal
  let colour = checkers pln.colour point
  let shine = pln.shine
  in (point, normal, colour, shine)

-- Cast a single ray into the scene.  In the Accelerate formulation,
-- this is done by a bounded loop that is fully unrolled at
-- compile-time.  Since we don't have a powerful meta-language, we
-- have to actually implement the loop.  I had to mangle things a
-- little bit since the original formulation is recursive, so I'm not
-- sure the optics are exactly the same.  We are also able to escape
-- early, in case a ray fails to collide with anyting.
let trace_ray (limit: i32) ({spheres,planes}: objects) (lights: lights)
              (ambient: argb.colour) (orig_point: position) (orig_dir: direction) =
  let (_, refl_colour,_,_,_) =
    loop (i, refl_colour, point, dir, visibility) =
         (0, argb.black, orig_point, orig_dir, 1.0) while i < limit do
    let (hit_s, dist_s, s) = cast_ray_sphere spheres point dir
    let (hit_p, dist_p, p) = cast_ray_plane planes point dir
    in if !(hit_s || hit_p) then (limit, refl_colour, point, dir, visibility) else
    -- Ray hit an object.
    let next_s = hit_sphere s dist_s point dir
    let next_p = hit_plane p dist_p point dir

    -- Does the sphere or plane count?
    let (point, normal, colour, shine) =
      if dist_s < dist_p then next_s else next_p

    -- Determine reflection angle.
    let newdir = dir vec3.- vec3.scale (2.0 * vec3.dot normal dir) normal

    -- Determine direct lighting at this point.
    let direct = apply_lights {spheres=spheres,planes=planes} lights point normal

    -- Total lighting is direct plus ambient
    let lighting = argb.add_linear direct ambient

    let light_in = argb.scale (argb.mult lighting colour) (1.0-shine)

    let light_out = argb.mix (1.0-visibility) refl_colour visibility light_in

    in (i+1,
        light_out,
        point,
        newdir,
        visibility * shine)
  let (r, g, b, _) = argb.to_rgba refl_colour
  in [u8.f32(r*255), u8.f32(g*255), u8.f32(b*255)]

type world = {objects: objects, lights: lights}

entry empty_world: world = {objects={spheres = [], planes = []}, lights = ([]: lights)}

entry add_sphere ({objects={spheres, planes}, lights}: world)
                 (x: f32) (y: f32) (z: f32)
                 (radius: f32)
                 (colour: argb.colour)
                 (shine: f32): (world, i32) =
  ({objects={spheres = spheres ++ [{position={x, y, z}, radius, colour, shine}],
             planes},
    lights},
   length spheres)

entry set_sphere_positions [n]
                 ({objects={spheres, planes}, lights}: world)
                 (xs: [n]f32) (ys: [n]f32) (zs: [n]f32) : world =
  let update x y z {position=_, radius, colour, shine} =
    { position = {x,y,z}, radius, colour, shine }
  in ({objects={spheres = map4 update xs ys zs spheres,
                planes},
       lights})

entry set_sphere_radius ({objects={spheres, planes}, lights}: world)
                        (i: i32) (radius: f32) : world =
  let update {position, radius=_, colour, shine} =
    { position, radius, colour, shine }
  in ({objects={spheres = copy spheres with [i] <- update spheres[i],
                planes},
       lights})

entry add_plane ({objects={spheres, planes}, lights}: world)
                (pos_x: f32) (pos_y: f32) (pos_z: f32)
                (norm_x: f32) (norm_y: f32) (norm_z: f32)
                (colour: argb.colour)
                (shine: f32): (world, i32) =
  ({objects={spheres,
             planes = planes ++ [{position={x=pos_x, y=pos_y, z=pos_z},
                                  normal={x=norm_x, y=norm_y, z=norm_z},
                                  colour,
                                  shine}]},
    lights},
   length planes)

entry add_light ({objects, lights}: world)
                (x: f32) (y: f32) (z: f32)
                (colour: argb.colour) (intensity: f32): world =
  {objects,
   lights = lights ++ [{position={x, y, z}, colour, intensity}] }

entry render ({objects, lights}: world)
             (sizeX: i32) (sizeY: i32) (fov: i32)
             (eye_pos_X: f32) (eye_pos_Y: f32) (eye_pos_Z: f32)
             (eye_dir_A: f32) (eye_dir_B: f32)
             (ambient: argb.colour) (ambient_intensity: f32)
             (limit: i32) : [sizeY][sizeX][3]u8 =
  let (r,g,b,_) = argb.to_rgba ambient
  let ambient = argb.from_rgba (r*ambient_intensity) (g*ambient_intensity) (b*ambient_intensity) 1.0
  let eye_pos = {x=eye_pos_X, y=eye_pos_Y, z=eye_pos_Z}
  let eye_dir = {x=f32.cos eye_dir_A * f32.cos eye_dir_B,
                 y=f32.sin eye_dir_B,
                 z=f32.sin eye_dir_A * f32.cos eye_dir_B}
  let eye_rays = cast_view_rays sizeX sizeY fov eye_dir
  in map (map (trace_ray limit objects lights ambient eye_pos)) eye_rays
