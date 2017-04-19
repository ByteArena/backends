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
        if(this.x == 0 && this.y == 0) {
            return 0;
        }

        let angle = Math.atan2(this.y, this.x);

        // Quart de tour à gauche
        angle = Math.PI/2.0 - angle;

        if (angle < 0) {
            angle += 2 * Math.PI;
        }

        return angle;
    }

    static fromArray(arr) {
        return new Vector2(arr[0], arr[1]);
    }
}

if(typeof module !== "undefined") module.exports = Vector2;
