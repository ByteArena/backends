module.exports = class Vector2 {

    constructor(x = null, y = null) {
        this.x = x || 0;
        this.y = y || this.x;
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

    static fromArray(arr) {
        return new Vector2(arr[0], arr[1]);
    }
}
