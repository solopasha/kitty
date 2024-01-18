Performance
===================

The main goals for |kitty| performance are user perceived latency while typing
and "smoothness" while scrolling as well as CPU usage. |kitty| tries hard to
find an optimum balance for these. To that end it keeps a cache of each
rendered glyph in video RAM so that font rendering is not a bottleneck.
Interaction with child programs takes place in a separate thread from
rendering, to improve smoothness. Parsing of the byte stream is done using
`vector CPU instructions
<https://en.wikipedia.org/wiki/Single_instruction,_multiple_data>`__ for
maximum performance. Updates to the screen typically require sending just a few
bytes to the GPU.

There are two config options you can tune to adjust the performance,
:opt:`repaint_delay` and :opt:`input_delay`. These control the artificial delays
introduced into the render loop to reduce CPU usage. See
:ref:`conf-kitty-performance` for details. See also the :opt:`sync_to_monitor`
option to further decrease latency at the cost of some `screen tearing
<https://en.wikipedia.org/wiki/Screen_tearing>`__ while scrolling.

Benchmarks
-------------

Measuring terminal emulator performance is fairly subtle, there are three main
axes on which performance is measured: Energy usage for typical tasks,
Keyboard to screen latency, and throughput (processing large amounts of data).

Keyboard to screen latency
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

This is measured either with dedicated hardware, or software such as `Typometer
<https://pavelfatin.com/typometer/>`__. Third party measurements comparing
kitty with other terminal emulators on various systems show kitty has best in
class keyboard to screen latency.

`Hardware based measurement on macOS
<https://thume.ca/2020/05/20/making-a-latency-tester/>`__ show that kitty and
Apple's Terminal.app share the crown for best latency. These
measurements were done with :opt:`input_delay` at its default value of ``3 ms``
which means kitty's actual numbers would be even lower.

`Typometer based measurements on Linux
<https://github.com/kovidgoyal/kitty/issues/2701#issuecomment-911089374>`__
show that kitty has far and away the best latency of the terminals tested.

Throughput
^^^^^^^^^^^^^^^^

kitty has a builtin kitten to measure throughput, it works by dumping large
amounts of data of different types into the tty device and measuring how fast
the terminal parses and responds to it. The measurements below were taken with
the same font, font size and window size for all terminals, and default
settings, on the same computer. They clearly show kitty has the fastest
throughput. To run the tests yourself, run ``kitten __benchmark__`` in the
terminal emulator you want to test, where the kitten binary is part of the
kitty install.

The numbers are megabytes per second of data that the terminal
processes. Measurements were taken under Linux/X11 with a ``AMD Ryzen 7 PRO
5850U``. Entries are in order of decreasing performance. kitty is clearly
faster than the rest.

================   ======  ======= ===== ====== =======
Terminal           ASCII   Unicode CSI   Images Average
================   ======  ======= ===== ====== =======
kitty 0.33         88.3    74.1    41.2  235.8  109.85
alacritty 0.13.1   43.1    46.5    32.5  94.1   54.05
wezterm 20230712   16.4    26.0    11.1  140.5  48.5
xterm 389          47.7    18.3    0.6   56.3   30.72
konsole 23.08.04   25.2    37.7    23.6  23.4   27.48
================   ======  ======= ===== ====== =======


.. note::

   By default, the benchmark kitten suppresses actual rendering, to better focus
   on parser speed, you can pass it the ``--render`` flag to not suppress
   rendering. However, modern terminals typically render asynchronously,
   therefore the numbers are not really useful for comparison. However, even
   with rendering enabled kitty is still much faster than all the rest. For
   brevity those numbers are not included.

.. note::

   gnome-terminal is left out as I could not get it to start on my system
   and foot is left out as it does not run under X11.

.. note::

   konsole and xterm dont support the `Synchronized update
   <https://gitlab.com/gnachman/iterm2/-/wikis/synchronized-updates-spec>`__
   escape code used to suppress rendering, if and when they gain support for it
   their numbers are likely to improve by ``20 - 30%``.


Energy usage
^^^^^^^^^^^^^^^^^

Sadly, I do not have the infrastructure to measure actual energy usage so CPU
usage will have to stand in for it. Here are some CPU usage numbers for the
task of scrolling a file continuously in :program:`less`. The CPU usage is for
the terminal process and X together and is measured using :program:`htop`. The
measurements are taken at the same font and window size for all terminals on a
``Intel(R) Core(TM) i7-4820K CPU @ 3.70GHz`` CPU with a ``Advanced Micro
Devices, Inc. [AMD/ATI] Cape Verde XT [Radeon HD 7770/8760 / R7 250X]`` GPU.

==============   =========================
Terminal         CPU usage (X + terminal)
==============   =========================
|kitty|          6 - 8%
xterm            5 - 7% (but scrolling was extremely janky)
termite          10 - 13%
urxvt            12 - 14%
gnome-terminal   15 - 17%
konsole          29 - 31%
==============   =========================

As you can see, |kitty| uses much less CPU than all terminals, except xterm, but
its scrolling "smoothness" is much better than that of xterm (at least to my,
admittedly biased, eyes).

Instrumenting kitty
-----------------------

You can generate detailed per-function performance data using
`gperftools <https://github.com/gperftools/gperftools>`__. Build |kitty| with
``make profile``. Run kitty and perform the task you want to analyse, for
example, scrolling a large file with :program:`less`. After you quit, function
call statistics will be printed to STDOUT and you can use tools like
*KCachegrind* for more detailed analysis.
