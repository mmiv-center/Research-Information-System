var step = 0;

function draw() {
    var val = jQuery('#exampleInput').val();
    var canvas = document.getElementById('canvas');
    if (canvas.getContext) {
        var ctx = canvas.getContext('2d');
        var width = ctx.canvas.width;
        var height = ctx.canvas.height;
        ctx.clearRect(0, 0, width, height);
        ctx.save();
        var scale = 20;
        
        ctx.beginPath();
        ctx.lineWidth = 2;
        ctx.strokeStyle = "rgb(66,44,255)";
        
        var x = 0;
        var y = 0;
        var amplitude = val;
        var frequency = 20;
        while (x < width) {
                y = height/2 + amplitude * Math.sin(x/frequency);
                ctx.lineTo(x, y);
                x = x + 1;
        }
        ctx.stroke();
        ctx.restore();
        
        step += 4;
    }
    window.requestAnimationFrame(draw);
}

jQuery(document).ready(function() {
    // react to events on the user interface
    jQuery('#compute').on('click', function() {
	// get a value from the interface
	var val = jQuery('#exampleInput').val();
	jQuery('#result').append(val+0);
    });

    window.requestAnimationFrame(draw);
});
