class Vector2 {

    constructor(x = null, y = null) {
        this.x = x || 0;
        this.y = y !== null ? y : this.x;
    }

    mag(mag = null) {
        if(mag === null) return Math.sqrt(this.magSq());
        return this.normalize().mult(mag)
    }

    magSq() {
        return (this.x * this.x + this.y * this.y);
    }

    limit(max) {
        var mSq = this.magSq();
        if(mSq > max*max) {
            this.div(Math.sqrt(mSq)); //normalize it
            this.mult(max);
        }
        return this;
    }

    div(v) {
        if(typeof v === 'number') {
            this.x /= v;
            this.y /= v;
        } else {
            this.x /= v.x;
            this.y /= v.y;
        }
        return this;
    }

    mult(v) {
        if(typeof v === 'number') {
            this.x *= v;
            this.y *= v;
        } else {
            this.x *= v.x;
            this.y *= v.y;
        }
        return this;
    }

    normalize() {
        const mag = this.mag();
        if(mag > 0) this.div(mag)
        return this;
    }

    toArray(precision = null) {
        if(precision === null) return [this.x, this.y];
        return [parseFloat(this.x.toFixed(precision)), parseFloat(this.y.toFixed(precision))];
    }

    clone() {
        return new Vector2(this.x, this.y);
    }

    add(v) {
        if(typeof v === 'number') {
            this.x += v;
            this.y += v;
        } else {
            this.x += v.x;
            this.y += v.y
        }

        return this;
    }

    sub(v) {
        if(typeof v === 'number') {
            this.x -= v;
            this.y -= v;
        } else {
            this.x -= v.x;
            this.y -= v.y
        }

        return this;
    }

    setAngle(radians) {
        const mag = this.mag()
	    this.x = Math.sin(radians) * mag;
	    this.y = Math.cos(radians) * mag;
        return this;
    }

    angle() {
        return Math.atan2(this.y, this.x);
    }

    rotate(rad) {
        const newHeading = this.angle() + rad;
        const mag = this.mag();
        this.x = Math.cos(newHeading) * mag;
        this.y = Math.sin(newHeading) * mag;
        return this;
    }

    // returns the normals to the line segment determined by the vector (as aligned on (0, 0))
    normals() {
        // dx=x2-x1 and dy=y2-y1, then the normals are (-dy, dx) and (dy, -dx).
        const x1 = 0, y1 = 0;
        const x2 = this.x, y2 = this.y;

        const dx = x2 - x1; const dy = y2 - y1;

        return [new Vector2(-dy, dx), new Vector2(dy, -dx)];
    }

    dot(v) {
	    return this.x*v.x + this.y*v.y;
    }

    cross(v) {
	    return this.x*v.y - this.y*v.x;
    }

    static intersectionWithLineSegment(p, p2, q, q2) {

        const r = p2.clone().sub(p);
        const s = q2.clone().sub(q);

        const rxs = r.cross(s);
        const qpxr = q.clone().sub(p).cross(r);

        // If r x s = 0 and (q - p) x r = 0, then the two lines are collinear.
        if (isZero(rxs) && isZero(qpxr)) {
            // 1. If either  0 <= (q - p) * r <= r * r or 0 <= (p - q) * s <= * s
            // then the two lines are overlapping,
            const qSubPTimesR = q.clone().sub(p).dot(r);
            const pSubQTimesS = p.clone().sub(q).dot(s);
            const rSquared = r.dot(r);
            const sSquared = s.dot(s);

            if ((qSubPTimesR >= 0 && qSubPTimesR <= rSquared) || (pSubQTimesS >= 0 && pSubQTimesS <= sSquared)) {
                return {
                    intersection: null,
                    intersects: true,
                    colinear: true,
                    parallel: true
                };
            }

            // 2. If neither 0 <= (q - p) * r = r * r nor 0 <= (p - q) * s <= s * s
            // then the two lines are collinear but disjoint.
            // No need to implement this expression, as it follows from the expression above.
            return {
                intersection: null,
                intersects: false,
                colinear: true,
                parallel: true
            };
        }

        // 3. If r x s = 0 and (q - p) x r != 0, then the two lines are parallel and non-intersecting.
        if(isZero(rxs) && !isZero(qpxr)) {
            return {
                intersection: null,
                intersects: false,
                colinear: false,
                parallel: true
            };
        }

        const t = q.clone().sub(p).cross(s) / rxs;
        const u = q.clone().sub(p).cross(r) / rxs;

        // 4. If r x s != 0 and 0 <= t <= 1 and 0 <= u <= 1
        // the two line segments meet at the point p + t r = q + u s.
        if(!isZero(rxs) && (0 <= t && t <= 1) && (0 <= u && u <= 1)) {
            // We can calculate the intersection point using either t or u.
            return {
                intersection: p.clone().add(r.mult(t)),
                intersects: true,
                colinear: false,
                parallel: false
            };
        }

        // 5. Otherwise, the two line segments are not parallel but do not intersect.
        return {
            intersection: null,
            intersects: false,
            colinear: false,
            parallel: true
        };
    }

    static fromArray(arr) {
        return new Vector2(arr[0], arr[1]);
    }
}

function isZero(val) {
    return val < 0.0000001;
}

if(typeof module !== "undefined") module.exports = Vector2;
